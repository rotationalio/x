/*
Package v1 stores secrets as opaque ciphertext using a version 1 envelope: per-row metadata,
an ephemeral X25519 exchange, HKDF-derived keys, a wrapped per-row data key, and an inner AES-256-GCM
payload. Construct a [Vault] with [New], passing your long-term X25519 private key, [storage.Storage], and
[identifier.Identifier]. Namespace is chosen per operation on [Vault.Store], [Vault.Retrieve], and related
methods; each sealed row binds the namespace you pass into authenticated metadata. Opening under a
different namespace than the row was sealed for fails with [v1errs.ErrNamespaceMismatch].

The library does not register a process-wide singleton; keep the [Vault] returned from [New] for
the lifetime of your wrapping key and storage wiring.
*/
package v1

// This file defines the [Vault] interface, [New], and the envelope seal/open helpers that connect
// crypto to [storage.Storage] and [identifier.Identifier]. Wire structs live in [models]; wire
// sizes and magic live in [constants].

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"errors"
	"io"

	"go.rtnl.ai/x/vault"
	verrors "go.rtnl.ai/x/vault/errors"
	"go.rtnl.ai/x/vault/identifier"
	"go.rtnl.ai/x/vault/keys"
	"go.rtnl.ai/x/vault/storage"
	"go.rtnl.ai/x/vault/v1/constants"
	v1errs "go.rtnl.ai/x/vault/v1/errors"
	vaultgcm "go.rtnl.ai/x/vault/v1/gcm"
	"go.rtnl.ai/x/vault/v1/models"
)

//=============================================================================
// Vault
//=============================================================================

// sealedVault implements [Vault] using an X25519 private key.
type sealedVault struct {
	priv     *ecdh.PrivateKey
	template models.Meta // namespace is set on each Store/Update call
	st       storage.Storage
	id       identifier.Identifier
}

// Ensure sealedVault implements [vault.Vault].
var _ vault.Vault = (*sealedVault)(nil)

// New constructs a [Vault] for the v1 envelope suite from an X25519 private key.
// Nil storage or identifier yields [verrors.ErrInvalidNewArgs]; a nil key yields [verrors.ErrNilPrivateKey];
// a non-X25519 curve yields [verrors.ErrInvalidWrappingKey]. Building the metadata template from the key
// can also fail (for instance [v1errs.ErrMetaKeyIDTooLarge]) if the public key id exceeds wire limits.
func New(priv *ecdh.PrivateKey, st storage.Storage, id identifier.Identifier) (vault.Vault, error) {
	if st == nil || id == nil {
		return nil, verrors.ErrInvalidNewArgs
	}
	meta, err := models.MetaFromPrivKey(priv)
	if err != nil {
		return nil, err
	}
	return &sealedVault{priv: priv, template: meta, st: st, id: id}, nil
}

// Store encrypts plaintext and persists a new row. A nil vault returns [verrors.ErrNilVault].
// [identifier.Identifier.New] failures are joined with [verrors.ErrInvalidIdentifier]. Problems while sealing
// (metadata, randomness, or crypto) return that envelope error directly. If [storage.Storage.Create]
// fails—duplicate key if the minted id collides, or any other backend error—the error is joined
// with [verrors.ErrStorage].
func (v *sealedVault) Store(ctx context.Context, namespace string, plaintext []byte) (id string, err error) {
	// Return error if vault receiver is nil.
	if v == nil {
		return "", verrors.ErrNilVault
	}

	// Generate a new row ID.
	id, err = v.id.New()
	if err != nil {
		return "", errors.Join(verrors.ErrInvalidIdentifier, err)
	}

	// Seal (encrypt) the plaintext into a wire-ready format.
	wire, err := v.sealPlaintext(namespace, plaintext)
	if err != nil {
		return "", err
	}

	// Write the sealed row to storage for this namespace and ID.
	if err := v.st.Create(ctx, namespace, id, wire); err != nil {
		return "", errors.Join(verrors.ErrStorage, err)
	}

	// Return the generated ID on success.
	return id, nil
}

// Retrieve loads ciphertext for (namespace, id) and decrypts it to plaintext.
// A nil vault returns [verrors.ErrNilVault]. Bad ids return [verrors.ErrInvalidIdentifier] joined with the parse
// error. If [storage.Storage.Get] fails—often because the row is absent—the error is joined with
// [verrors.ErrStorage] and typically chains [verrors.ErrNotFound]. After a blob is loaded, corrupt or mismatched
// ciphertext surfaces as wire errors from this package or decrypt failures from package gcm
// (see [gcm]), including [v1errs.ErrNamespaceMismatch], without wrapping in [verrors.ErrStorage].
func (v *sealedVault) Retrieve(ctx context.Context, namespace, id string) (plaintext []byte, err error) {
	// Validate that the receiver is non-nil.
	if v == nil {
		return nil, verrors.ErrNilVault
	}

	// Parse and validate the given row ID.
	if err := v.id.Parse(id); err != nil {
		return nil, errors.Join(verrors.ErrInvalidIdentifier, err)
	}

	// Retrieve the sealed row from storage for this namespace and ID.
	wire, err := v.st.Get(ctx, namespace, id)
	if err != nil {
		return nil, errors.Join(verrors.ErrStorage, err)
	}

	// Decrypt (open) the ciphertext and return the plaintext.
	return v.openCiphertext(namespace, wire)
}

// Update replaces plaintext for an existing row. A nil vault returns [verrors.ErrNilVault].
// Parse failures join [verrors.ErrInvalidIdentifier]. Re-sealing can fail the same way as during [Store].
// If [storage.Storage.Replace] fails—most often [verrors.ErrNotFound] when the row does not exist—the error is
// joined with [verrors.ErrStorage].
func (v *sealedVault) Update(ctx context.Context, namespace, id string, plaintext []byte) error {

	// Validate that the receiver is non-nil.
	if v == nil {
		return verrors.ErrNilVault
	}

	// Parse and validate the given row ID.
	if err := v.id.Parse(id); err != nil {
		return errors.Join(verrors.ErrInvalidIdentifier, err)
	}

	// Seal (encrypt) the provided plaintext into a wire-ready format.
	wire, err := v.sealPlaintext(namespace, plaintext)
	if err != nil {
		return err
	}

	// Replace the sealed row in storage for this namespace and ID.
	if err := v.st.Replace(ctx, namespace, id, wire); err != nil {
		return errors.Join(verrors.ErrStorage, err)
	}

	// Update successful, return nil error.
	return nil
}

// CompareAndSwap reads the row, checks that decrypted plaintext equals currentPlain, seals
// newPlain, then replaces storage only if the ciphertext is still the bytes that were read.
// Nil receivers return [verrors.ErrNilVault]; bad ids join [verrors.ErrInvalidIdentifier]. A failed read joins
// [verrors.ErrStorage]. Opening the current ciphertext returns decrypt and wire errors the same way as
// [Retrieve], without [verrors.ErrStorage]. A mismatch with currentPlain yields [verrors.ErrWrongCurrent] and
// leaves storage unchanged. Sealing newPlain can fail like [Store]. Finally, if another writer
// changed the row or deleted it, [storage.Storage.CompareAndSwap] fails with [verrors.ErrCASFailed] or
// [verrors.ErrNotFound] (among others), joined with [verrors.ErrStorage].
func (v *sealedVault) CompareAndSwap(ctx context.Context, namespace, id string, currentPlain, newPlain []byte) error {
	// Validate that the receiver is non-nil.
	if v == nil {
		return verrors.ErrNilVault
	}

	// Parse and validate the given row ID.
	if err := v.id.Parse(id); err != nil {
		return errors.Join(verrors.ErrInvalidIdentifier, err)
	}

	// Retrieve the current sealed row from storage for this namespace and ID.
	oldWire, err := v.st.Get(ctx, namespace, id)
	if err != nil {
		return errors.Join(verrors.ErrStorage, err)
	}

	// Decrypt the current row to obtain the plaintext.
	plain, err := v.openCiphertext(namespace, oldWire)
	if err != nil {
		return err
	}

	// Caller-supplied expected plaintext must match decrypted row exactly.
	if !bytes.Equal(plain, currentPlain) {
		return verrors.ErrWrongCurrent
	}

	// Seal (encrypt) the new plaintext.
	newWire, err := v.sealPlaintext(namespace, newPlain)
	if err != nil {
		return err
	}

	// Attempt to replace wire only if storage still holds the blob we opened (CAS operation).
	if err := v.st.CompareAndSwap(ctx, namespace, id, oldWire, newWire); err != nil {
		return errors.Join(verrors.ErrStorage, err)
	}

	// CAS operation was successful.
	return nil
}

// MoveNamespace decrypts under oldNamespace, re-seals under newNamespace, writes the new row,
// then deletes the old one so the secret is bound to the destination namespace's metadata.
// Equal namespaces are a no-op. A nil vault returns [verrors.ErrNilVault]; parse failures join
// [verrors.ErrInvalidIdentifier]. [storage.Storage.Get] or [storage.Storage.Create] errors join [verrors.ErrStorage]. Opening the
// old blob or sealing for the new namespace propagates envelope errors without [verrors.ErrStorage], as
// in [Retrieve] and [Store]. If the new row was created but deleting the old key fails, the error
// joins [verrors.ErrMoveNamespaceIncomplete], [verrors.ErrStorage], and the delete failure—the copy in the new
// namespace may remain alongside the old row until the application reconciles.
func (v *sealedVault) MoveNamespace(ctx context.Context, oldNamespace, newNamespace, id string) error {
	// Return error if vault receiver is nil.
	if v == nil {
		return verrors.ErrNilVault
	}

	// If old and new namespace are the same, do nothing.
	if oldNamespace == newNamespace {
		return nil
	}

	// Validate the provided ID.
	if err := v.id.Parse(id); err != nil {
		return errors.Join(verrors.ErrInvalidIdentifier, err)
	}

	// Fetch the sealed row from the old namespace.
	oldWire, err := v.st.Get(ctx, oldNamespace, id)
	if err != nil {
		return errors.Join(verrors.ErrStorage, err)
	}

	// Decrypt row contents.
	plain, err := v.openCiphertext(oldNamespace, oldWire)
	if err != nil {
		return err
	}

	// Reseal the same plaintext using the new namespace to bind new AAD/metadata.
	newWire, err := v.sealPlaintext(newNamespace, plain)
	if err != nil {
		return err
	}

	// Create the row in the new namespace with the resealed blob.
	if err := v.st.Create(ctx, newNamespace, id, newWire); err != nil {
		return errors.Join(verrors.ErrStorage, err)
	}

	// Only after successfully creating in newNamespace, delete from the old.
	// This ensures callers never see both rows missing except on error.
	if err := v.st.Delete(ctx, oldNamespace, id); err != nil {
		return errors.Join(verrors.ErrMoveNamespaceIncomplete, verrors.ErrStorage, err)
	}

	return nil
}

// Delete removes a row after validating the id. A nil vault returns [verrors.ErrNilVault]; parse
// failures join [verrors.ErrInvalidIdentifier]. If [storage.Storage.Delete] returns an error, it is joined with
// [verrors.ErrStorage]. Conforming storage treats a missing row as success, so deleting an absent id still
// yields nil.
func (v *sealedVault) Delete(ctx context.Context, namespace, id string) error {
	// Return error if vault receiver is nil.
	if v == nil {
		return verrors.ErrNilVault
	}

	// Validate the provided ID.
	if err := v.id.Parse(id); err != nil {
		return errors.Join(verrors.ErrInvalidIdentifier, err)
	}

	// Attempt to delete the row from storage.
	if err := v.st.Delete(ctx, namespace, id); err != nil {
		return errors.Join(verrors.ErrStorage, err)
	}

	// Successfully deleted or row was already absent.
	return nil
}

//=============================================================================
// Envelope seal and open
//=============================================================================

// sealPlaintext builds row metadata for namespace, seals plaintext, and returns
// the v1 wire blob.
func (v *sealedVault) sealPlaintext(namespace string, plaintext []byte) ([]byte, error) {
	// Generate a fresh, random Data Encryption Key (DEK) for this row.
	dek := make([]byte, constants.DEKBytes)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return nil, verrors.ErrSealFailed
	}
	defer keys.Zero(dek)

	// Generate a unique nonce for the inner AEAD encryption.
	var innerNonce [constants.InnerNonceBytes]byte
	if _, err := io.ReadFull(rand.Reader, innerNonce[:]); err != nil {
		return nil, verrors.ErrSealFailed
	}

	// Generate ephemeral X25519 key for envelope wrapping.
	ephPriv, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, verrors.ErrSealFailed
	}

	// Generate a nonce for the envelope (wrapping) AEAD.
	var wrapNonce [constants.WrapNonceBytes]byte
	if _, err := io.ReadFull(rand.Reader, wrapNonce[:]); err != nil {
		return nil, verrors.ErrSealFailed
	}

	// Seal the plaintext and return the complete envelope using fresh keys/nonces.
	return v.sealPlaintextWith(namespace, plaintext, dek, innerNonce, ephPriv, wrapNonce)
}

// sealPlaintextWith seals plaintext using fixed DEK, nonces, and ephemeral key.
// NOTE: this is separated from sealPlaintext so we can generate fixed golden
// vector tests easily.
func (v *sealedVault) sealPlaintextWith(namespace string, plaintext, dek []byte, innerNonce [constants.InnerNonceBytes]byte, ephPriv *ecdh.PrivateKey, wrapNonce [constants.WrapNonceBytes]byte) ([]byte, error) {
	defer keys.Zero(dek)

	// Prepare per-row metadata, copying the template and injecting this operation's namespace.
	row, err := v.template.WithNamespace(namespace)
	if err != nil {
		return nil, err
	}

	// Marshal metadata to binary for authenticated encryption and binding.
	metaRaw, err := row.MarshalBinary()
	if err != nil {
		return nil, err
	}

	// Build the AEAD used to encrypt the user's data (inner payload).
	innerAEAD, err := vaultgcm.NewInnerAEAD(dek)
	if err != nil {
		return nil, err
	}

	// Encrypt the plaintext (sealing the data and binding metadata as AAD).
	nonce, payload, err := vaultgcm.SealInnerWithNonce(innerAEAD, metaRaw, plaintext, innerNonce)
	if err != nil {
		return nil, err
	}
	body := models.Inner{Nonce: nonce, Payload: payload}

	// ECDH: derive a shared secret from ephemeral private and long-term public key.
	shared, err := ephPriv.ECDH(v.priv.PublicKey())
	if err != nil {
		return nil, err
	}

	// Stretch the shared secret into an envelope wrapping key.
	wrapKey, err := vaultgcm.DeriveWrapKey(shared)
	if err != nil {
		return nil, err
	}
	defer keys.Zero(wrapKey)

	// Build AEAD for the envelope (to wrap the DEK).
	wrapAEAD, err := vaultgcm.NewWrapAEAD(wrapKey)
	if err != nil {
		return nil, err
	}

	// Prepare the ephemeral public key to include in the wire format.
	var ephPub [constants.X25519PubBytes]byte
	copy(ephPub[:], ephPriv.PublicKey().Bytes())

	// Encrypt (wrap) the DEK for transport, sealing it with envelope AEAD and AAD (metadata).
	dekWire, err := vaultgcm.SealWrappedDEKWithNonce(ephPub, wrapAEAD, vaultgcm.WrapAAD(metaRaw), dek, wrapNonce)
	if err != nil {
		return nil, err
	}
	dekEnv := models.DekEnvelope{Pub: dekWire.Pub, Nonce: dekWire.Nonce, Payload: dekWire.Payload}

	// Assemble the complete sealed wire, including all envelope components.
	sealed := models.Sealed{
		FormatVersion: constants.PackageVersion,
		Meta:          row,
		Dek:           dekEnv,
		Body:          body,
	}

	// Marshal the final sealed row as a single wire blob.
	return sealed.MarshalBinary()
}

// openCiphertext parses wire, unwraps keys, verifies plaintext, and checks
// namespace matches requestedNS.
func (v *sealedVault) openCiphertext(requestedNS string, wire []byte) ([]byte, error) {
	// Unmarshal the sealed wire into the Sealed structure.
	var msg models.Sealed
	if err := msg.UnmarshalBinary(wire); err != nil {
		return nil, err
	}

	// Ensure that the namespace matches the one requested.
	if msg.Meta.Namespace != requestedNS {
		return nil, v1errs.ErrNamespaceMismatch
	}

	// Marshal the row metadata for use as associated data.
	metaRaw, err := msg.Meta.MarshalBinary()
	if err != nil {
		return nil, err
	}

	// Reconstruct the ephemeral public key for ECDH.
	epub, err := ecdh.X25519().NewPublicKey(msg.Dek.Pub[:])
	if err != nil {
		return nil, verrors.ErrDecrypt
	}

	// Perform ECDH with our private key and the ephemeral public key.
	shared, err := v.priv.ECDH(epub)
	if err != nil {
		return nil, verrors.ErrDecrypt
	}

	// Derive the wrapping key from the shared secret.
	wrapKey, err := vaultgcm.DeriveWrapKey(shared)
	if err != nil {
		return nil, err
	}
	defer keys.Zero(wrapKey)

	// Build AEAD for unwrapping the DEK.
	wrapAEAD, err := vaultgcm.NewWrapAEAD(wrapKey)
	if err != nil {
		return nil, err
	}

	// Unwrap and authenticate the DEK using the AEAD and metadata.
	wrapped := vaultgcm.WrappedDEK{Pub: msg.Dek.Pub, Nonce: msg.Dek.Nonce, Payload: msg.Dek.Payload}
	dek, err := vaultgcm.OpenWrappedDEK(wrapAEAD, vaultgcm.WrapAAD(metaRaw), wrapped)
	if err != nil {
		return nil, err
	}
	defer keys.Zero(dek)

	// Build AEAD for decrypting the inner ciphertext.
	innerAEAD, err := vaultgcm.NewInnerAEAD(dek)
	if err != nil {
		return nil, err
	}

	// Open and verify the inner ciphertext with the decrypted DEK and metadata.
	plain, err := vaultgcm.OpenInner(innerAEAD, metaRaw, msg.Body.Nonce, msg.Body.Payload)
	if err != nil {
		return nil, err
	}

	return plain, nil
}

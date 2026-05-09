// Package vault encrypts secrets at rest using AES-256-GCM and pluggable
// [Storage] backends.
//
// Each ciphertext is authenticated with the secret's namespace as GCM
// additional data (AAD). If a stored blob is read under a different namespace
// string, decryption fails (authentication error). Use [Vault.MoveNamespace]
// to relocate a secret so ciphertext is re-sealed with the new namespace AAD.
//
// Optional process-wide initialization uses [Init] and [SnapshotVault].
// Callers may also use [New] for isolated instances or tests. UTF-8 and JSON
// helpers live in subpackages [go.rtnl.ai/x/vault/stringvault] and
// [go.rtnl.ai/x/vault/jsonvault].
//
// See [go.rtnl.ai/x/vault/README.md] for more details.
package vault

import (
	"bytes"
	"context"
	"crypto/cipher"
	"errors"
	"sync"
)

//=============================================================================
// Vault type and process singleton
//=============================================================================

// Vault encrypts plaintext with AES-256-GCM and delegates persistence to
// [Storage].
type Vault struct {
	aead cipher.AEAD
	st   Storage
	id   Identifier
}

// New constructs a [Vault] with a 32-byte AES-256 key, storage, and identifier.
func New(key []byte, st Storage, id Identifier) (*Vault, error) {
	if st == nil || id == nil {
		return nil, ErrInvalidNewArgs
	}

	// We will only support 32-byte keys for new vaults (AES-256).
	if len(key) != 32 {
		return nil, ErrInvalidKeyLength
	}

	// Create the AEAD cipher.
	aead, err := newAEAD(key)
	if err != nil {
		return nil, errors.Join(ErrAEADSetup, err)
	}

	return &Vault{aead: aead, st: st, id: id}, nil
}

// Process-wide vault singleton state.
var (
	vaultMu      sync.RWMutex
	vaultInst    *Vault
	vaultOnce    sync.Once
	vaultInitErr error
)

// Init installs the process-wide vault singleton. The first call wins; later
// calls are no-ops and return the same error as the first attempt if
// initialization failed.
func Init(key []byte, st Storage, id Identifier) error {
	vaultOnce.Do(func() {
		v, err := New(key, st, id)
		if err != nil {
			vaultInitErr = err
			return
		}
		vaultMu.Lock()
		vaultInst = v
		vaultMu.Unlock()
	})
	return vaultInitErr
}

// SnapshotVault returns the initialized [*Vault] and whether initialization
// succeeded.
func SnapshotVault() (*Vault, bool) {
	vaultMu.RLock()
	defer vaultMu.RUnlock()
	return vaultInst, vaultInst != nil
}

// Reset clears the singleton for tests only.
func Reset() {
	vaultMu.Lock()
	defer vaultMu.Unlock()
	vaultInst = nil
	vaultInitErr = nil
	vaultOnce = sync.Once{}
}

//=============================================================================
// User operations
//=============================================================================

// Store encrypts plain and creates a new secret; the id is minted by [Identifier.New].
func (v *Vault) Store(ctx context.Context, namespace string, plain []byte) (id string, err error) {
	// Mint a new identifier.
	id, err = v.id.New()
	if err != nil {
		return "", errors.Join(ErrInvalidIdentifier, err)
	}

	// Seal the plaintext with the namespace as AAD.
	ct, err := seal(v.aead, []byte(namespace), plain)
	if err != nil {
		return "", err
	}

	// Create the new row in storage.
	if err := v.st.Create(ctx, namespace, id, ct); err != nil {
		return "", errors.Join(ErrStorage, err)
	}

	return id, nil
}

// Update blindly replaces the ciphertext for an existing secret.
func (v *Vault) Update(ctx context.Context, namespace, id string, newPlain []byte) error {
	// Parse the id for validity.
	if err := v.id.Parse(id); err != nil {
		return errors.Join(ErrInvalidIdentifier, err)
	}

	// Encrypt the new plaintext with namespace as AAD.
	ct, err := seal(v.aead, []byte(namespace), newPlain)
	if err != nil {
		return err
	}

	// Replace the ciphertext for this secret in storage.
	if err := v.st.Replace(ctx, namespace, id, ct); err != nil {
		return errors.Join(ErrStorage, err)
	}
	return nil
}

// AtomicUpdate replaces the secret only if the decrypted plaintext equals
// currentPlain. [ErrWrongCurrent] means the caller's expected plaintext did
// not match storage. [ErrCASFailed] means another writer changed the row
// between read and compare-and-swap.
func (v *Vault) AtomicUpdate(ctx context.Context, namespace, id string, currentPlain, newPlain []byte) error {
	// Parse the id for validity.
	if err := v.id.Parse(id); err != nil {
		return errors.Join(ErrInvalidIdentifier, err)
	}

	// Retrieve the current ciphertext for the secret.
	oldCipher, err := v.st.Get(ctx, namespace, id)
	if err != nil {
		return errors.Join(ErrStorage, err)
	}

	// Decrypt the existing ciphertext.
	plain, err := open(v.aead, []byte(namespace), oldCipher)
	if err != nil {
		return err
	}

	// Check if the plaintext matches the expected 'currentPlain'.
	if !bytes.Equal(plain, currentPlain) {
		return ErrWrongCurrent
	}

	// Encrypt the new plaintext.
	newCipher, err := seal(v.aead, []byte(namespace), newPlain)
	if err != nil {
		return err
	}

	// Atomically replace if and only if the old ciphertext matches.
	if err := v.st.CompareAndSwap(ctx, namespace, id, oldCipher, newCipher); err != nil {
		return errors.Join(ErrStorage, err)
	}
	return nil
}

// Retrieve decrypts and returns plaintext.
func (v *Vault) Retrieve(ctx context.Context, namespace, id string) ([]byte, error) {
	// Parse the id for validity.
	if err := v.id.Parse(id); err != nil {
		return nil, errors.Join(ErrInvalidIdentifier, err)
	}

	// Fetch the ciphertext from storage.
	ct, err := v.st.Get(ctx, namespace, id)
	if err != nil {
		return nil, errors.Join(ErrStorage, err)
	}

	// Decrypt and return the plaintext.
	return open(v.aead, []byte(namespace), ct)
}

// MoveNamespace moves a secret from fromNamespace to toNamespace by decrypting,
// re-sealing with the new namespace, creating the new entry, and deleting the
// old. If namespaces are equal, does nothing. If delete fails after create,
// both may exist.
func (v *Vault) MoveNamespace(ctx context.Context, fromNamespace, toNamespace, id string) error {
	// If the namespaces are equal, do nothing.
	if fromNamespace == toNamespace {
		return nil
	}

	// Parse the id for validity.
	if err := v.id.Parse(id); err != nil {
		return errors.Join(ErrInvalidIdentifier, err)
	}

	// Retrieve the old ciphertext from the source namespace.
	oldCipher, err := v.st.Get(ctx, fromNamespace, id)
	if err != nil {
		return errors.Join(ErrStorage, err)
	}

	// Decrypt to plaintext from the source namespace's AAD.
	plain, err := open(v.aead, []byte(fromNamespace), oldCipher)
	if err != nil {
		return err
	}

	// Reseal plaintext with the new namespace as additional authenticated data.
	newCipher, err := seal(v.aead, []byte(toNamespace), plain)
	if err != nil {
		return err
	}

	// Insert the resealed ciphertext in the new namespace.
	if err := v.st.Create(ctx, toNamespace, id, newCipher); err != nil {
		return errors.Join(ErrStorage, err)
	}

	// Remove the secret from the old namespace.
	if err := v.st.Delete(ctx, fromNamespace, id); err != nil {
		return errors.Join(ErrMoveNamespaceIncomplete, ErrStorage, err)
	}

	return nil
}

// Delete removes the secret; missing rows are OK (idempotent).
func (v *Vault) Delete(ctx context.Context, namespace, id string) error {
	// Parse the id for validity.
	if err := v.id.Parse(id); err != nil {
		return errors.Join(ErrInvalidIdentifier, err)
	}

	// Remove the secret from storage.
	if err := v.st.Delete(ctx, namespace, id); err != nil {
		return errors.Join(ErrStorage, err)
	}

	return nil
}

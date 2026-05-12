package v1_test

// Golden wire contract: a fixed sealed row must still open with the same long-term key.
//
// Regenerate goldenSealedV1Hex after intentional wire/crypto changes:
//
//	go test -tags=emitgolden ./vault/v1 -run TestEmitGoldenV1Wire -v
//
// Then update the hex constants in this file from the test log (see golden_emit_test.go).

import (
	"context"
	"crypto/ecdh"
	"encoding/hex"
	"testing"

	"go.rtnl.ai/x/assert"
	v1 "go.rtnl.ai/x/vault/v1"
	"go.rtnl.ai/x/vault/v1/constants"
	verrors "go.rtnl.ai/x/vault/v1/errors"
	"go.rtnl.ai/x/vault/v1/identifier"
	"go.rtnl.ai/x/vault/v1/models"
	"go.rtnl.ai/x/vault/v1/storage"
)

// Long-term X25519 private key bytes that produced goldenSealedV1Hex (32-byte scalar encoding).
const goldenLongTermPrivHex = "6d4cbdc47df6b0d6c534aa744ab4ba0fdfe90edf9d28a9101229b69eec79f80a"

// Full v1 sealed row wire (magic VLT1, meta, DEK envelope, inner) for namespace "golden-ns"
// and plaintext "hello-golden", built with ExportTestBuildSealedRow and fixed nonces/DEK/ephemeral key.
const goldenSealedV1Hex = "564c543101002d010120501e2366b36fe14d71a926f2858d155536ce5a173d69f7731b4a6cd5cd07d96509676f6c64656e2d6e73471118583b237cbaadd25a5ba98d0ddd79ff5ceaa3cd31097e426dda877497370708090a0b0c0d0e0f101112f707569cfca3daa51c06264ecd8aab1c53a1e85dcc60d19370bbd9b1402f33c8f993b9a653ae7ae1660aa88b9f1b573505060708090a0b0c0d0e0f1056b695ea0ba7256c4fa01dff854cafa49e54873426b638646a560ea8"

// TestGoldenV1Contract checks a frozen sealed blob still unmarshals and decrypts with current v1 code.
func TestGoldenV1Contract(t *testing.T) {

	// Set expected plaintext and namespace for validation.
	wantPlain := []byte("hello-golden")
	const goldenNS = "golden-ns"

	// Decode the hardcoded X25519 private key.
	privBytes, err := hex.DecodeString(goldenLongTermPrivHex)
	assert.Ok(t, err)
	priv, err := ecdh.X25519().NewPrivateKey(privBytes)
	assert.Ok(t, err)

	// Decode golden sealed row wire bytes.
	wire, err := hex.DecodeString(goldenSealedV1Hex)
	assert.Ok(t, err)

	// Unmarshal the sealed row and perform basic invariants check.
	var msg models.Sealed
	assert.Ok(t, msg.UnmarshalBinary(wire))
	assert.Equal(t, constants.PackageVersion, msg.FormatVersion) // Check row format version.
	assert.Equal(t, goldenNS, msg.Meta.Namespace)                // Ensure namespace matches.

	// Store sealed row in an in-memory Storage for retrieval.
	st := storage.NewMemStorage()
	ctx := context.Background()
	const rowID = "0123456789abcdef0123456789abcdef" // Use fixed hex row ID for test.
	assert.Ok(t, st.Create(ctx, goldenNS, rowID, wire))

	// Create a new vault instance using the decoded private key and memory storage.
	v, err := v1.New(priv, st, identifier.HexIdentifier{})
	assert.Ok(t, err)

	// Retrieve and decrypt, then compare plaintext.
	got, err := v.Retrieve(ctx, goldenNS, rowID)
	assert.Ok(t, err)
	assert.Equal(t, wantPlain, got)
}

// TestGoldenV1Contract_tamperedWireFailsDecrypt flips one sealed byte and expects decrypt failure.
func TestGoldenV1Contract_tamperedWireFailsDecrypt(t *testing.T) {
	// Decode the hex-encoded long-term private key and sealed row.
	privBytes, err := hex.DecodeString(goldenLongTermPrivHex)
	assert.Ok(t, err)

	// Create a new X25519 private key from the decoded bytes.
	priv, err := ecdh.X25519().NewPrivateKey(privBytes)
	assert.Ok(t, err)

	// Decode the golden sealed row wire bytes to use as the ciphertext.
	wire, err := hex.DecodeString(goldenSealedV1Hex)
	assert.Ok(t, err)

	// Tamper with the last byte of the wire to simulate corruption.
	wire[len(wire)-1] ^= 0xff

	// Set up storage, context, and a fixed row ID for the test case.
	st := storage.NewMemStorage()
	ctx := context.Background()
	const rowID = "0123456789abcdef0123456789abcdef"

	// Store the tampered wire blob in storage under the test namespace and row ID.
	assert.Ok(t, st.Create(ctx, "golden-ns", rowID, wire))

	// Initialize a new Vault backed by our private key and the test memory storage.
	v, err := v1.New(priv, st, identifier.HexIdentifier{})
	assert.Ok(t, err)

	// Attempt to retrieve and decrypt using the tampered wire;
	// expect decryption to fail with ErrDecrypt.
	_, err = v.Retrieve(ctx, "golden-ns", rowID)
	assert.ErrorIs(t, err, verrors.ErrDecrypt)
}

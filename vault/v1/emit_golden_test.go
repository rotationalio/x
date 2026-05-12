//go:build emitgolden

package v1_test

// Regenerate golden constants in golden_test.go:
//
//	go test -tags=emitgolden ./vault/v1 -run TestEmitGoldenV1Wire -v
//
// Copy WIRE_HEX and (if you change keys) PRIV_HEX from the log into golden_test.go.

import (
	"crypto/ecdh"
	"encoding/hex"
	"testing"

	"go.rtnl.ai/x/assert"
	v1 "go.rtnl.ai/x/vault/v1"
	"go.rtnl.ai/x/vault/v1/constants"
)

// TestEmitGoldenV1Wire prints fixed hex for the long-term key and golden sealed wire (build tag emitgolden).
// Use it to refresh constants in [golden_test.go] after intentional wire or crypto changes.
func TestEmitGoldenV1Wire(t *testing.T) {
	privBytes, err := hex.DecodeString("6d4cbdc47df6b0d6c534aa744ab4ba0fdfe90edf9d28a9101229b69eec79f80a")
	assert.Ok(t, err)
	ephBytes, err := hex.DecodeString("c6c30cbb51f3dbb6147eaff18eeacda816c5d2da20840479ccfe7346a4f7c97c")
	assert.Ok(t, err)
	priv, err := ecdh.X25519().NewPrivateKey(privBytes)
	assert.Ok(t, err)
	ephPriv, err := ecdh.X25519().NewPrivateKey(ephBytes)
	assert.Ok(t, err)
	dek := make([]byte, 32)
	for i := range dek {
		dek[i] = byte(i + 3)
	}
	var innerNonce [constants.InnerNonceBytes]byte
	for i := range innerNonce {
		innerNonce[i] = byte(i + 5)
	}
	var wrapNonce [constants.WrapNonceBytes]byte
	for i := range wrapNonce {
		wrapNonce[i] = byte(i + 7)
	}
	wire, err := v1.ExportTestBuildSealedRow(priv, "golden-ns", []byte("hello-golden"), dek, innerNonce, ephPriv, wrapNonce)
	assert.Ok(t, err)
	t.Logf("WIRE_HEX=%s", hex.EncodeToString(wire))
}

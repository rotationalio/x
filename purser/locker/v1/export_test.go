package v1

// Export_test exposes selected internals to tests in package v1_test.

import (
	"crypto/ecdh"

	"go.rtnl.ai/x/vault"
	"go.rtnl.ai/x/vault/v1/constants"
	"go.rtnl.ai/x/vault/v1/models"
)

// ExportTestBuildSealedRow builds a v1 sealed wire blob using fixed DEK, inner and wrap nonces,
// and an ephemeral X25519 keypair. It matches [sealedVault.sealPlaintext] output for the same
// inputs and exists for generating golden vector tests.
func ExportTestBuildSealedRow(priv *ecdh.PrivateKey, namespace string, plaintext, dek []byte, innerNonce [constants.InnerNonceBytes]byte, ephPriv *ecdh.PrivateKey, wrapNonce [constants.WrapNonceBytes]byte) ([]byte, error) {
	dekCopy := append([]byte(nil), dek...)
	meta, err := models.MetaFromPrivKey(priv)
	if err != nil {
		return nil, err
	}
	v := &sealedVault{priv: priv, template: meta, st: nil, id: nil}
	return v.sealPlaintextWith(namespace, plaintext, dekCopy, innerNonce, ephPriv, wrapNonce)
}

// NilSealedVault is a typed-nil [*sealedVault] as [Vault] for nil-receiver contract tests in package v1_test.
var NilSealedVault vault.Vault = (*sealedVault)(nil)

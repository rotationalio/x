package locker

// Key construction: FromPassword, FromSeed, FromPKCS8, FromKey, or New in locker.go.

import (
	"crypto/ecdh"
	"crypto/x509"

	"go.rtnl.ai/x/purser"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/keyring"
)

const (
	// SeedBytes is the required input size for FromSeed (32 bytes for X25519).
	SeedBytes = 32
)

// FromPassword stretches a password with Argon2id and returns a ready-to-use v1 Locker.
// The caller must persist salt alongside the user/device record for re-derivation.
func FromPassword(password, salt []byte, p keyring.Params) (purser.Locker, error) {
	seed, err := keyring.Derive(password, salt, p, SeedBytes)
	if err != nil {
		return nil, err
	}
	defer purser.Zero(seed)
	return FromSeed(seed)
}

// FromSeed maps a derived 32-byte seed to a v1 Locker (X25519 long-term key).
func FromSeed(seed []byte) (purser.Locker, error) {
	if len(seed) != SeedBytes {
		return nil, perrors.ErrInvalidSeed
	}
	priv, err := ecdh.X25519().NewPrivateKey(seed)
	if err != nil {
		return nil, err
	}
	return New(priv)
}

// FromPKCS8 parses a PKCS#8 private key and returns a v1 Locker (X25519 only).
func FromPKCS8(der []byte) (purser.Locker, error) {
	key, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, err
	}
	return FromKey(key)
}

// FromKey accepts an X25519 *ecdh.PrivateKey and returns a v1 Locker.
func FromKey(key any) (purser.Locker, error) {
	priv, ok := key.(*ecdh.PrivateKey)
	if !ok || priv == nil {
		return nil, perrors.ErrInvalidWrappingKey
	}
	return New(priv)
}

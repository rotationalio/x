package v1

// Key construction for locker/v1: password-based (FromPassword), seed-based (FromSeed),
// or bring-your-own X25519 key (New in locker.go).

import (
	"crypto/ecdh"

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

	priv, err := FromSeed(seed)
	if err != nil {
		return nil, err
	}
	return New(priv)
}

// FromSeed maps a derived 32-byte seed to a long-term X25519 private key for locker/v1.
func FromSeed(seed []byte) (*ecdh.PrivateKey, error) {
	if len(seed) != SeedBytes {
		return nil, perrors.ErrInvalidSeed
	}
	return ecdh.X25519().NewPrivateKey(seed)
}

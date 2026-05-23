/*
Package registry dispatches locker construction and wire ParseKeyID by edition.

Pass [purser.EditionV1] as the edition argument to FromPassword and FromSeed when using the
root package constant.
*/
package registry

import (
	"slices"

	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/keyring/kdf"
	"go.rtnl.ai/x/purser/locker"
	lockerv1 "go.rtnl.ai/x/purser/locker/v1"
	constv1 "go.rtnl.ai/x/purser/locker/v1/constants"
	"go.rtnl.ai/x/purser/wire"
)

// Editions returns supported edition strings sorted lexicographically.
func Editions() []string {
	out := make([]string, len(registered))
	for i, e := range registered {
		out[i] = e.edition
	}
	slices.Sort(out)
	return out
}

// FromSeed returns a locker for edition using a derived seed.
func FromSeed(edition string, seed []byte) (locker.Locker, error) {
	h, ok := hooks(edition)
	if !ok || h.fromSeed == nil {
		return nil, perrors.ErrUnsupportedLockerVersion
	}
	return h.fromSeed(seed)
}

// FromPassword stretches password and salt with Argon2id for edition.
func FromPassword(edition string, password, salt []byte, p kdf.Params) (locker.Locker, error) {
	h, ok := hooks(edition)
	if !ok || h.fromPassword == nil {
		return nil, perrors.ErrUnsupportedLockerVersion
	}
	return h.fromPassword(password, salt, p)
}

// FromPKCS8 parses PKCS#8 DER, trying each built-in edition until one succeeds.
func FromPKCS8(der []byte) (locker.Locker, error) {
	for _, e := range registered {
		if e.fromPKCS8 == nil {
			continue
		}
		if lck, err := e.fromPKCS8(der); err == nil {
			return lck, nil
		}
	}
	return nil, perrors.ErrInvalidWrappingKey
}

// FromKey accepts a private key, trying each built-in edition until one succeeds.
func FromKey(key any) (locker.Locker, error) {
	for _, e := range registered {
		if e.fromKey == nil {
			continue
		}
		if lck, err := e.fromKey(key); err == nil {
			return lck, nil
		}
	}
	return nil, perrors.ErrInvalidWrappingKey
}

// ParseKeyID reads the sealing key id from purser wire without decrypting.
func ParseKeyID(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < wire.PreambleBytes || string(ciphertext[:wire.MagicLen]) != wire.Magic {
		return nil, perrors.ErrUnrecognizedCiphertext
	}
	ver := ciphertext[wire.VersionOffset]
	for _, e := range registered {
		if e.parseKeyID == nil {
			continue
		}
		if e.wireVersion != ver {
			continue
		}
		kid, err := e.parseKeyID(ciphertext)
		if err == nil {
			return kid, nil
		}
	}
	return nil, perrors.ErrUnrecognizedCiphertext
}

// editionHooks holds the wiring for a single edition.
type editionHooks struct {
	wireVersion uint8
	edition     string

	fromSeed     func([]byte) (locker.Locker, error)
	fromPassword func(password, salt []byte, p kdf.Params) (locker.Locker, error)
	fromPKCS8    func([]byte) (locker.Locker, error)
	fromKey      func(any) (locker.Locker, error)
	parseKeyID   func([]byte) ([]byte, error)
}

// registered holds the built-in edition hooks.
var registered = []editionHooks{
	{
		wireVersion:  constv1.Version,
		edition:      constv1.Edition,
		fromSeed:     lockerv1.FromSeed,
		fromPassword: lockerv1.FromPassword,
		fromPKCS8:    lockerv1.FromPKCS8,
		fromKey:      lockerv1.FromKey,
		parseKeyID:   lockerv1.ParseKeyID,
	},
}

// hooks returns built-in edition wiring for edition.
func hooks(edition string) (editionHooks, bool) {
	for _, e := range registered {
		if e.edition == edition {
			return e, true
		}
	}
	return editionHooks{}, false
}

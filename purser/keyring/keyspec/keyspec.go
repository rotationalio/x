/*
Package keyspec builds one-time locker inputs and returns a [locker.Locker] via [registry].

Single-import clients use [purser.KeySpec], [purser.NewPassword], [purser.NewSeed], [purser.NewPKCS8],
and [purser.NewPrivateKey] (see purser aliases.go).
*/
package keyspec

import (
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/internal/memzero"
	"go.rtnl.ai/x/purser/keyring/kdf"
	"go.rtnl.ai/x/purser/keyring/registry"
	"go.rtnl.ai/x/purser/locker"
	lockerv1 "go.rtnl.ai/x/purser/locker/v1"
)

// KeySpec holds sensitive key material for a single Register or Locker call.
type KeySpec struct {
	kind    kind
	edition string

	password []byte
	salt     []byte
	kdf      kdf.Params
	seed     []byte
	pkcs8    []byte
	privKey  any
}

// NewPassword returns a KeySpec for password-based locker derivation. edition
// must be non-empty.
func NewPassword(password, salt []byte, p kdf.Params, edition string) (*KeySpec, error) {
	if len(password) == 0 {
		return nil, perrors.ErrNilPassword
	}
	if len(salt) != kdf.SaltBytes {
		return nil, perrors.ErrInvalidSalt
	}
	if err := requireEdition(edition); err != nil {
		return nil, err
	}
	return &KeySpec{
		kind:     kindPassword,
		edition:  edition,
		password: append([]byte(nil), password...),
		salt:     append([]byte(nil), salt...),
		kdf:      p,
	}, nil
}

// NewSeed returns a KeySpec for a pre-derived seed. edition must be non-empty.
func NewSeed(seed []byte, edition string) (*KeySpec, error) {
	if len(seed) != lockerv1.SeedBytes {
		return nil, perrors.ErrInvalidSeed
	}
	if err := requireEdition(edition); err != nil {
		return nil, err
	}
	return &KeySpec{
		kind:    kindSeed,
		edition: edition,
		seed:    append([]byte(nil), seed...),
	}, nil
}

// NewPKCS8 returns a KeySpec for PKCS#8 private key bytes. edition must be non-empty.
func NewPKCS8(der []byte, edition string) (*KeySpec, error) {
	if len(der) == 0 {
		return nil, perrors.ErrInvalidKeySpec
	}
	if err := requireEdition(edition); err != nil {
		return nil, err
	}
	return &KeySpec{
		kind:    kindPKCS8,
		edition: edition,
		pkcs8:   append([]byte(nil), der...),
	}, nil
}

// NewPrivateKey returns a KeySpec wrapping an in-process private key (X25519 for v1). edition must be non-empty.
func NewPrivateKey(key any, edition string) (*KeySpec, error) {
	if key == nil {
		return nil, perrors.ErrInvalidKeySpec
	}
	if err := requireEdition(edition); err != nil {
		return nil, err
	}
	return &KeySpec{
		kind:    kindPrivateKey,
		edition: edition,
		privKey: key,
	}, nil
}

// Locker builds a locker via registry and wipes sensitive fields (single-use).
func (s *KeySpec) Locker() (locker.Locker, error) {
	if s == nil || s.kind == kindUnset {
		return nil, perrors.ErrInvalidKeySpec
	}
	defer s.wipe()

	var (
		lck locker.Locker
		err error
	)
	switch s.kind {
	case kindPassword:
		lck, err = registry.FromPassword(s.edition, s.password, s.salt, s.kdf)
	case kindSeed:
		lck, err = registry.FromSeed(s.edition, s.seed)
	case kindPKCS8:
		lck, err = registry.FromPKCS8(s.pkcs8)
		if err == nil && lck != nil && lck.Edition() != s.edition {
			err = perrors.ErrUnsupportedLockerVersion
			lck = nil
		}
	case kindPrivateKey:
		lck, err = registry.FromKey(s.privKey)
		if err == nil && lck != nil && lck.Edition() != s.edition {
			err = perrors.ErrUnsupportedLockerVersion
			lck = nil
		}
	default:
		return nil, perrors.ErrInvalidKeySpec
	}
	return lck, err
}

// requireEdition rejects an empty locker edition string.
func requireEdition(edition string) error {
	if edition == "" {
		return perrors.ErrInvalidEdition
	}
	return nil
}

type kind int

const (
	kindUnset kind = iota
	kindPassword
	kindSeed
	kindPKCS8
	kindPrivateKey
)

// wipe clears sensitive fields after Locker consumes the spec.
func (s *KeySpec) wipe() {
	if s == nil {
		return
	}
	memzero.Zero(s.password)
	memzero.Zero(s.salt)
	memzero.Zero(s.seed)
	memzero.Zero(s.pkcs8)
	s.password = nil
	s.salt = nil
	s.seed = nil
	s.pkcs8 = nil
	s.privKey = nil
	s.kind = kindUnset
}

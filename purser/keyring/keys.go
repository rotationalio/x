// Package keys derives Argon2id key material from passwords with explicit [Params], generates
// salts with [RandSalt], and maps a 32-byte derived secret to an X25519 private key with [FromSeed].
//
// Two ready-made profiles match [RFC 9106] Argon2id recommendations: [DefaultParams] is the **first**
// recommended option (t=1, 2 GiB memory) for environments that can afford the RAM per hash;
// [MemoryConstrainedParams] is the **second** recommended option (t=3, 64 MiB) for memory-constrained
// environments (containers, small VMs, many concurrent derives). We follow the RFC profiles for
// **maximum hardness** (memory-hard settings from the standard). The [OWASP Password Storage Cheat Sheet]
// suggests different parameter sets (generally lower memory and tuned for practical login latency);
// use those or any other tuning by constructing your own [Params] and passing them to [Derive].
// Ensure [Params.Threads] is at least 1 or [argon2.IDKey] panics.
//
// [RFC 9106]: https://www.rfc-editor.org/rfc/rfc9106
// [argon2.IDKey]: https://pkg.go.dev/golang.org/x/crypto/argon2#IDKey
// [OWASP Password Storage Cheat Sheet]: https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html#argon2id
package keys

import (
	"crypto/ecdh"
	"crypto/rand"
	"io"

	verrors "go.rtnl.ai/x/vault/errors"
	"golang.org/x/crypto/argon2"
)

const (
	SaltBytes       = 16
	DerivedKeyBytes = 32
)

// Params holds Argon2id tuning parameters.
type Params struct {
	Iterations uint32 // t parameter
	MemoryKiB  uint32 // m parameter
	Threads    uint8  // parallelism / lanes, ≥ 1
}

// RFC 9106 Argon2id first recommended profile: t=1, 2 GiB memory (see package comment).
const (
	rfc9106FirstRecommendedTime      uint32 = 1
	rfc9106FirstRecommendedMemoryKiB uint32 = 2 * 1024 * 1024 // 2 GiB
)

// RFC 9106 Argon2id second recommended profile: t=3, 64 MiB (2^16 KiB) for memory-constrained environments.
const (
	rfc9106SecondRecommendedTime      uint32 = 3
	rfc9106SecondRecommendedMemoryKiB uint32 = 65536 // 64 MiB
)

// defaultParallelism is the lane count used by [DefaultParams] and
// [MemoryConstrainedParams].
const defaultParallelism uint8 = 1

// RandSalt returns a new random salt for Argon2id (16 bytes).
func RandSalt() ([]byte, error) {
	b := make([]byte, SaltBytes)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, verrors.ErrRandSalt
	}
	return b, nil
}

// DefaultParams returns the RFC 9106 **first** recommended Argon2id profile: t=1, 2 GiB memory,
// parallelism 1. Expect roughly 2 GiB peak memory per [Derive] call (plus overhead).
func DefaultParams() Params {
	return Params{
		Iterations: rfc9106FirstRecommendedTime,
		MemoryKiB:  rfc9106FirstRecommendedMemoryKiB,
		Threads:    defaultParallelism,
	}
}

// MemoryConstrainedParams returns the RFC 9106 **second** recommended Argon2id profile: t=3, 64 MiB
// memory, parallelism 1. Use when 2 GiB per hash is impractical (e.g. tight container limits).
func MemoryConstrainedParams() Params {
	return Params{
		Iterations: rfc9106SecondRecommendedTime,
		MemoryKiB:  rfc9106SecondRecommendedMemoryKiB,
		Threads:    defaultParallelism,
	}
}

// Derive runs Argon2id and returns outLen bytes of key material.
func Derive(password, salt []byte, p Params, outLen int) ([]byte, error) {
	if password == nil {
		return nil, verrors.ErrNilPassword
	}
	if len(salt) != SaltBytes {
		return nil, verrors.ErrInvalidSalt
	}
	if outLen <= 0 {
		return nil, verrors.ErrInvalidOut
	}
	return argon2.IDKey(password, salt, p.Iterations, p.MemoryKiB, p.Threads, uint32(outLen)), nil
}

// Zero overwrites b with zeros.
func Zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// FromSeed maps a derived 32-byte seed to a long-term X25519 private key.
func FromSeed(seed []byte) (*ecdh.PrivateKey, error) {
	if len(seed) != DerivedKeyBytes {
		return nil, verrors.ErrInvalidSeed
	}
	return ecdh.X25519().NewPrivateKey(seed)
}

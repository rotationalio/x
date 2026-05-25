/*
Package kdf provides Argon2id Derive, RandSalt, and RFC 9106 parameter profiles.

Single-import clients use [purser.RandSalt], [purser.Params], [purser.DefaultParams], and
[purser.MemoryConstrainedParams] (see purser aliases.go). Call Derive and SaltBytes from this package.
*/
package kdf

import (
	"crypto/rand"
	"io"

	perrors "go.rtnl.ai/x/purser/errors"
	"golang.org/x/crypto/argon2"
)

const (
	// SaltBytes is the required salt length for Derive (16 bytes, matching RFC 9106 examples).
	SaltBytes = 16
)

// Params holds Argon2id tuning parameters.
type Params struct {
	Iterations uint32
	MemoryKiB  uint32
	Threads    uint8
}

const (
	rfc9106FirstRecommendedTime       uint32 = 1
	rfc9106FirstRecommendedMemoryKiB  uint32 = 2 * 1024 * 1024
	rfc9106SecondRecommendedTime      uint32 = 3
	rfc9106SecondRecommendedMemoryKiB uint32 = 65536
)

const defaultParallelism uint8 = 1

// RandSalt returns a new random salt for Argon2id (16 bytes).
func RandSalt() ([]byte, error) {
	b := make([]byte, SaltBytes)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, perrors.ErrRandSalt
	}
	return b, nil
}

// DefaultParams returns the RFC 9106 first recommended Argon2id profile.
var DefaultParams = Params{
	Iterations: rfc9106FirstRecommendedTime,
	MemoryKiB:  rfc9106FirstRecommendedMemoryKiB,
	Threads:    defaultParallelism,
}

// MemoryConstrainedParams returns the RFC 9106 second recommended Argon2id profile.
var MemoryConstrainedParams = Params{
	Iterations: rfc9106SecondRecommendedTime,
	MemoryKiB:  rfc9106SecondRecommendedMemoryKiB,
	Threads:    defaultParallelism,
}

// Derive runs Argon2id and returns outLen bytes of key material.
func Derive(password, salt []byte, p Params, outLen int) ([]byte, error) {
	if password == nil {
		return nil, perrors.ErrNilPassword
	}
	if len(salt) != SaltBytes {
		return nil, perrors.ErrInvalidSalt
	}
	if outLen <= 0 {
		return nil, perrors.ErrInvalidOut
	}
	return argon2.IDKey(password, salt, p.Iterations, p.MemoryKiB, p.Threads, uint32(outLen)), nil
}

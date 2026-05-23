/*
Package registry holds per-edition locker constructors and version-dispatched From* helpers.

Locker/vN packages register in init via Register; importing go.rtnl.ai/x/purser links v1 by default.
*/
package registry

import (
	"slices"
	"sync"

	"go.rtnl.ai/x/purser/contract"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/keyring"
	"go.rtnl.ai/x/purser/wire"
)

// Hooks holds per-edition constructors registered by locker/vN init.
type Hooks struct {
	// WireVersion is the format-version byte after wire.Magic (e.g. v1 = 1). Zero skips version routing.
	WireVersion uint8

	FromSeed     func([]byte) (contract.Locker, error)
	FromPassword func(password, salt []byte, p keyring.Params) (contract.Locker, error)
	FromPKCS8    func([]byte) (contract.Locker, error)
	FromKey      func(any) (contract.Locker, error)
	ParseKeyID   func([]byte) ([]byte, error)
}

var (
	lockerMu       sync.RWMutex
	lockerEditions = map[string]Hooks{}
)

// Register wires locker/vN constructors into the registry.
// edition must match that package's constants.Edition. Called from locker/vN init only.
// Register returns ErrDuplicateLockerEdition or ErrDuplicateLockerWireVersion on conflict.
func Register(edition string, h Hooks) error {
	lockerMu.Lock()
	defer lockerMu.Unlock()
	if _, exists := lockerEditions[edition]; exists {
		return perrors.ErrDuplicateLockerEdition
	}
	if h.WireVersion != 0 {
		for _, existing := range lockerEditions {
			if existing.WireVersion == h.WireVersion {
				return perrors.ErrDuplicateLockerWireVersion
			}
		}
	}
	lockerEditions[edition] = h
	return nil
}

// Editions returns registered locker edition strings sorted lexicographically.
func Editions() []string {
	lockerMu.RLock()
	out := make([]string, 0, len(lockerEditions))
	for edition := range lockerEditions {
		out = append(out, edition)
	}
	lockerMu.RUnlock()
	slices.Sort(out)
	return out
}

// hooks returns hooks for edition under read lock.
func hooks(edition string) (Hooks, bool) {
	lockerMu.RLock()
	h, ok := lockerEditions[edition]
	lockerMu.RUnlock()
	return h, ok
}

// sortedEntries returns registered editions and hooks in lexicographic edition order.
func sortedEntries() []struct {
	edition string
	h       Hooks
} {
	editions := Editions()
	out := make([]struct {
		edition string
		h       Hooks
	}, len(editions))
	lockerMu.RLock()
	for i, edition := range editions {
		out[i] = struct {
			edition string
			h       Hooks
		}{edition: edition, h: lockerEditions[edition]}
	}
	lockerMu.RUnlock()
	return out
}

// FromSeed returns a locker for the given edition using a derived 32-byte seed.
func FromSeed(edition string, seed []byte) (contract.Locker, error) {
	h, ok := hooks(edition)
	if !ok || h.FromSeed == nil {
		return nil, perrors.ErrUnsupportedLockerVersion
	}
	return h.FromSeed(seed)
}

// FromPassword stretches password and salt with Argon2id and returns a locker for the edition.
func FromPassword(edition string, password, salt []byte, p keyring.Params) (contract.Locker, error) {
	h, ok := hooks(edition)
	if !ok || h.FromPassword == nil {
		return nil, perrors.ErrUnsupportedLockerVersion
	}
	return h.FromPassword(password, salt, p)
}

// FromPKCS8 parses a PKCS#8 private key, trying each registered edition in sorted order until one succeeds.
func FromPKCS8(der []byte) (contract.Locker, error) {
	for _, e := range sortedEntries() {
		if e.h.FromPKCS8 == nil {
			continue
		}
		if lck, err := e.h.FromPKCS8(der); err == nil {
			return lck, nil
		}
	}
	return nil, perrors.ErrInvalidWrappingKey
}

// FromKey accepts a private key value, trying each registered edition in sorted order until one succeeds.
func FromKey(key any) (contract.Locker, error) {
	for _, e := range sortedEntries() {
		if e.h.FromKey == nil {
			continue
		}
		if lck, err := e.h.FromKey(key); err == nil {
			return lck, nil
		}
	}
	return nil, perrors.ErrInvalidWrappingKey
}

// ParseKeyID reads the sealing key identifier from purser wire without decrypting.
// Requires wire.Magic and dispatches by the format-version byte to the matching edition.
func ParseKeyID(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < wire.PreambleBytes || string(ciphertext[:wire.MagicLen]) != wire.Magic {
		return nil, perrors.ErrUnrecognizedCiphertext
	}
	ver := ciphertext[wire.VersionOffset]
	for _, e := range sortedEntries() {
		if e.h.ParseKeyID == nil {
			continue
		}
		if e.h.WireVersion != 0 && e.h.WireVersion != ver {
			continue
		}
		kid, err := e.h.ParseKeyID(ciphertext)
		if err == nil {
			return kid, nil
		}
	}
	return nil, perrors.ErrUnrecognizedCiphertext
}

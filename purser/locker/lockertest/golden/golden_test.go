package golden_test

// Centralized table-driven golden tests for locker/vN implementations.
//
// Every locker version that ships a locker.go under purser/locker/vN must have a
// corresponding entry in cases(). TestLockerImplementationsCovered fails fast if any
// are missing. Fixtures are opaque .dat files in testdata/; deleting them triggers
// automatic regeneration on the next test run.
//
// All locker cases derive their key material from a single shared Argon2id password,
// salt, and parameter set. Each version's newLocker maps the derived seed to a
// version-specific private key so the seed size is standardized across versions.

import (
	crand "crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/purser"
	"go.rtnl.ai/x/purser/internal/nulllocker"
	"go.rtnl.ai/x/purser/keyring"
	v1 "go.rtnl.ai/x/purser/locker/v1"
)

//=============================================================================
// Shared Argon2id seed derivation
//=============================================================================

// goldenPassword and goldenSalt are fixed test credentials. All locker cases derive
// key material from the same Argon2id run so golden fixtures are reproducible.
var (
	goldenPassword = []byte("golden-test-password")
	goldenSalt     = []byte("golden-test-salt") // exactly keyring.SaltBytes (16)
)

// goldenParams uses minimal Argon2id work so tests stay fast.
var goldenParams = keyring.Params{Iterations: 1, MemoryKiB: 32, Threads: 1}

// goldenSeed derives the shared seed once. Every test case maps these bytes to a
// version-specific key via its own FromSeed (or equivalent).
func goldenSeed(t *testing.T) []byte {
	t.Helper()
	seed, err := keyring.Derive(goldenPassword, goldenSalt, goldenParams, v1.SeedBytes)
	assert.Ok(t, err)
	return seed
}

//=============================================================================
// Constants and types
//=============================================================================

const (
	testNamespace = "golden-ns"
	testPlaintext = "hello-golden"
)

// lockerCase binds a version name to a factory that produces a locker from the shared seed.
type lockerCase struct {
	version   string
	newLocker func(t *testing.T, seed []byte) (purser.Locker, error)
}

// cases returns the golden test table. Add a new entry when shipping a new locker version.
func cases() []lockerCase {
	return []lockerCase{
		{
			version: "v0-a",
			newLocker: func(t *testing.T, seed []byte) (purser.Locker, error) {
				return nulllocker.FromSeed(t, nulllocker.VariantA, seed)
			},
		},
		{
			version: "v0-b",
			newLocker: func(t *testing.T, seed []byte) (purser.Locker, error) {
				return nulllocker.FromSeed(t, nulllocker.VariantB, seed)
			},
		},
		{
			version: "v0-c",
			newLocker: func(t *testing.T, seed []byte) (purser.Locker, error) {
				return nulllocker.FromSeed(t, nulllocker.VariantC, seed)
			},
		},
		{
			version: "v1",
			newLocker: func(t *testing.T, seed []byte) (purser.Locker, error) {
				priv, err := v1.FromSeed(seed)
				if err != nil {
					return nil, err
				}
				return v1.New(priv)
			},
		},
	}
}

// fixturePath returns the on-disk path for a version's golden fixture.
func fixturePath(version string) string {
	return filepath.Join("testdata", version+".dat")
}

//=============================================================================
// Deterministic randomness for fixture generation
//=============================================================================

// deterministicReader emits a repeatable incrementing byte pattern for stable fixture generation.
type deterministicReader struct{ b byte }

// Read fills p with an incrementing byte sequence starting from r.b.
func (r *deterministicReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = r.b
		r.b++
	}
	return len(p), nil
}

//=============================================================================
// Tests
//=============================================================================

// TestLockerImplementationsCovered ensures every locker/vN directory that contains a
// locker.go has a corresponding golden test case. The null locker variants (v0-*) are
// under locker/internal/ and are handled separately; only top-level vN directories are
// checked here.
func TestLockerImplementationsCovered(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	assert.True(t, ok, "runtime caller unavailable")
	lockerDir := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))

	entries, err := os.ReadDir(lockerDir)
	assert.Ok(t, err)

	// Build a set of version strings from the test table.
	casesByVersion := map[string]struct{}{}
	for _, c := range cases() {
		casesByVersion[c.version] = struct{}{}
	}

	// Scan for directories matching vN (not v0-* which are null-locker variants).
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "v") {
			continue
		}
		if _, err := os.Stat(filepath.Join(lockerDir, name, "locker.go")); err != nil {
			continue
		}
		_, ok := casesByVersion[name]
		assert.True(t, ok, "locker implementation %q is missing a golden test case", name)
	}
}

// TestGoldenRoundTrip regenerates missing fixtures and verifies Open round-trips known
// plaintext. Delete a testdata/*.dat file and re-run tests to regenerate that fixture.
func TestGoldenRoundTrip(t *testing.T) {
	seed := goldenSeed(t)

	for _, tc := range cases() {
		t.Run(tc.version, func(t *testing.T) {
			fp := fixturePath(tc.version)

			// Generate the fixture if it doesn't exist yet.
			if _, err := os.Stat(fp); err != nil && os.IsNotExist(err) {
				lck, err := tc.newLocker(t, seed)
				assert.Ok(t, err)

				// Swap in a deterministic reader so the sealed blob is reproducible.
				orig := crand.Reader
				crand.Reader = &deterministicReader{}
				wire, sealErr := lck.Seal(testNamespace, []byte(testPlaintext))
				crand.Reader = orig
				assert.Ok(t, sealErr)

				assert.Ok(t, os.MkdirAll(filepath.Dir(fp), 0o755))
				assert.Ok(t, os.WriteFile(fp, wire, 0o644))
			}

			// Read the fixture and verify the round-trip.
			wire, err := os.ReadFile(fp)
			assert.Ok(t, err)
			lck, err := tc.newLocker(t, seed)
			assert.Ok(t, err)
			got, err := lck.Open(testNamespace, wire)
			assert.Ok(t, err)
			assert.Equal(t, []byte(testPlaintext), got)
		})
	}
}

// TestGoldenTamper ensures fixtures fail to open after tampering with the last byte.
func TestGoldenTamper(t *testing.T) {
	seed := goldenSeed(t)

	for _, tc := range cases() {
		t.Run(tc.version, func(t *testing.T) {
			wire, err := os.ReadFile(fixturePath(tc.version))
			assert.Ok(t, err)
			assert.True(t, len(wire) > 0, "empty fixture for %s", tc.version)

			// Flip the last byte so either AEAD auth or null-locker content changes.
			wire[len(wire)-1] ^= 0xff
			lck, err := tc.newLocker(t, seed)
			assert.Ok(t, err)
			_, err = lck.Open(testNamespace, wire)
			assert.Error(t, err, "expected decrypt failure for tampered %s fixture", tc.version)
		})
	}
}

// TestCasesBuildLockers sanity-checks every test case constructs a working locker
// that can seal and has a non-empty key ID.
func TestCasesBuildLockers(t *testing.T) {
	seed := goldenSeed(t)

	for _, tc := range cases() {
		t.Run(tc.version, func(t *testing.T) {
			lck, err := tc.newLocker(t, seed)
			assert.Ok(t, err)
			assert.NotNil(t, lck, "nil locker for %s", tc.version)
			assert.True(t, len(lck.KeyID()) > 0, "empty key id for %s", tc.version)
			_, err = lck.Seal(testNamespace, []byte("sanity"))
			assert.Ok(t, err)
		})
	}
}

// Example_fixturePath documents the fixture path convention.
func Example_fixturePath() {
	fmt.Println(fixturePath("v1"))
	// Output: testdata/v1.dat
}

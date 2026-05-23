package golden_test

// Centralized table-driven golden tests for locker/vN implementations.

import (
	crand "crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/purser/internal/nulllocker"
	"go.rtnl.ai/x/purser/keyring/kdf"
	"go.rtnl.ai/x/purser/locker"
	lockerv1 "go.rtnl.ai/x/purser/locker/v1"
)

//=============================================================================
// Tests
//=============================================================================

// TestLockerImplementationsCovered ensures every locker/vN directory that contains a
// locker.go has a corresponding golden test case.
func TestLockerImplementationsCovered(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	assert.True(t, ok, "runtime caller unavailable")
	lockerDir := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))

	entries, err := os.ReadDir(lockerDir)
	assert.Ok(t, err)

	casesByVersion := map[string]struct{}{}
	for _, c := range cases() {
		casesByVersion[c.version] = struct{}{}
	}

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

			if _, err := os.Stat(fp); err != nil && os.IsNotExist(err) {
				lck, err := tc.newLocker(t, seed)
				assert.Ok(t, err)

				orig := crand.Reader
				crand.Reader = &deterministicReader{}
				wire, sealErr := lck.Seal(testNamespace, []byte(testPlaintext))
				crand.Reader = orig
				assert.Ok(t, sealErr)

				assert.Ok(t, os.MkdirAll(filepath.Dir(fp), 0o755))
				assert.Ok(t, os.WriteFile(fp, wire, 0o644))
			}

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

			wire[len(wire)-1] ^= 0xff
			lck, err := tc.newLocker(t, seed)
			assert.Ok(t, err)
			_, err = lck.Open(testNamespace, wire)
			assert.Error(t, err, "expected decrypt failure for tampered %s fixture", tc.version)
		})
	}
}

// TestCasesBuildLockers sanity-checks every test case constructs a working locker.
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

// goldenPassword and goldenSalt are fixed test credentials for reproducible Argon2 runs.
var (
	goldenPassword = []byte("golden-test-password")
	goldenSalt     = []byte("golden-test-salt") // exactly kdf.SaltBytes (16)
)

// goldenParams uses minimal Argon2id work so tests stay fast.
var goldenParams = kdf.Params{Iterations: 1, MemoryKiB: 32, Threads: 1}

const (
	testNamespace = "golden-ns"
	testPlaintext = "hello-golden"
)

// lockerCase binds a version name to a factory that produces a locker from the shared seed.
type lockerCase struct {
	version   string
	newLocker func(t *testing.T, seed []byte) (locker.Locker, error)
}

// goldenSeed derives the shared seed used by all golden locker cases.
func goldenSeed(t *testing.T) []byte {
	t.Helper()
	seed, err := kdf.Derive(goldenPassword, goldenSalt, goldenParams, lockerv1.SeedBytes)
	assert.Ok(t, err)
	return seed
}

// cases returns the golden test table. Add a new entry when shipping a new locker version.
func cases() []lockerCase {
	return []lockerCase{
		{
			version: "v0-a",
			newLocker: func(t *testing.T, seed []byte) (locker.Locker, error) {
				return nulllocker.FromSeed(t, nulllocker.VariantA, seed)
			},
		},
		{
			version: "v0-b",
			newLocker: func(t *testing.T, seed []byte) (locker.Locker, error) {
				return nulllocker.FromSeed(t, nulllocker.VariantB, seed)
			},
		},
		{
			version: "v0-c",
			newLocker: func(t *testing.T, seed []byte) (locker.Locker, error) {
				return nulllocker.FromSeed(t, nulllocker.VariantC, seed)
			},
		},
		{
			version: "v1",
			newLocker: func(t *testing.T, seed []byte) (locker.Locker, error) {
				return lockerv1.FromSeed(seed)
			},
		},
	}
}

// fixturePath returns the on-disk path for a version's golden fixture.
func fixturePath(version string) string {
	return filepath.Join("testdata", version+".dat")
}

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

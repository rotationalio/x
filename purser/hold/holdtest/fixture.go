package holdtest

// Nulllocker wire fixtures for Hold conformance and integration tests.

import (
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/purser/contract"
	"go.rtnl.ai/x/purser/internal/nulllocker"
)

// fixtureSeed is the deterministic key-id seed shared by HoldConforms and Ciphertext.
var fixtureSeed = []byte("holdtest")

// fixtureVariant selects the nulllocker wire personality for conformance blobs.
const fixtureVariant = nulllocker.VariantA

// newFixtureLocker constructs the shared nulllocker used to seal conformance ciphertext.
func newFixtureLocker(tb testing.TB) contract.Locker {
	tb.Helper()
	lck, err := nulllocker.New(tb, fixtureVariant, fixtureSeed)
	assert.Ok(tb, err)
	return lck
}

// seal binds plaintext to namespace using the fixture locker.
func seal(lck contract.Locker, namespace string, plaintext []byte) ([]byte, error) {
	return lck.Seal(namespace, plaintext)
}

// Ciphertext returns nulllocker wire for plaintext bound to namespace.
// Use for Hold conformance and integration tests; not valid for registry.ParseKeyID (v1 only).
func Ciphertext(tb testing.TB, namespace string, plaintext []byte) []byte {
	tb.Helper()
	lck := newFixtureLocker(tb)
	wire, err := seal(lck, namespace, plaintext)
	assert.Ok(tb, err)
	return wire
}

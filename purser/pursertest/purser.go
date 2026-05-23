/*
Package pursertest provides a ready-made Purser for tests that uses the null locker
(no real crypto) wired through the real purser.New orchestration, so callers exercise
the full Purser → Keyring → Locker → Hold path without envelope encryption overhead.
*/
package pursertest

import (
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/purser"
	"go.rtnl.ai/x/purser/contract"
	"go.rtnl.ai/x/purser/hold"
	"go.rtnl.ai/x/purser/internal/nulllocker"
	"go.rtnl.ai/x/purser/keyring/memring"
)

// NewTestPurser returns a Purser backed by a null locker and the given hold. It uses the
// real purser.New orchestration so wrapper tests exercise the full code path.
func NewTestPurser(tb testing.TB, h hold.Hold) contract.Purser {
	tb.Helper()
	p, _ := NewTestPurserWithLocker(tb, h)
	return p
}

// NewTestPurserWithLocker returns a Purser and the underlying null Locker. The Locker is
// useful for tests that need to seal synthetic or corrupt content into the hold so it
// passes wire-format parsing on retrieve.
func NewTestPurserWithLocker(tb testing.TB, h hold.Hold) (contract.Purser, contract.Locker) {
	tb.Helper()

	assert.NotNil(tb, h, "pursertest: hold required")

	lck, err := nulllocker.New(tb, nulllocker.VariantA, []byte("test-seed"))
	assert.Ok(tb, err, "pursertest: nulllocker")

	kr, err := memring.New(lck)
	assert.Ok(tb, err, "pursertest: memring")

	p, err := purser.New(h, kr)
	assert.Ok(tb, err, "pursertest: purser.New")

	return p, lck
}

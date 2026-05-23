/*
Package pursertest provides a ready-made Purser for tests that uses the null locker
wired through [purser.New]. Production wiring uses [purser.NewMemHold], [purser.NewMemring],
[purser.NewPassword], and related symbols from purser aliases.go.
*/
package pursertest

import (
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/purser"
	"go.rtnl.ai/x/purser/hold"
	"go.rtnl.ai/x/purser/internal/nulllocker"
	"go.rtnl.ai/x/purser/keyring/memring"
	"go.rtnl.ai/x/purser/locker"
)

// NewTestPurser returns a Purser backed by a null locker and the given hold.
func NewTestPurser(tb testing.TB, h hold.Hold) *purser.Purser {
	tb.Helper()
	p, _ := NewTestPurserWithLocker(tb, h)
	return p
}

// NewTestPurserWithLocker returns a Purser and the underlying null Locker.
func NewTestPurserWithLocker(tb testing.TB, h hold.Hold) (*purser.Purser, locker.Locker) {
	tb.Helper()

	assert.NotNil(tb, h, "pursertest: hold required")

	lck, err := nulllocker.New(tb, nulllocker.VariantA, []byte("test-seed"))
	assert.Ok(tb, err, "pursertest: nulllocker")

	kr := memring.New()
	assert.Ok(tb, kr.SetDefault(lck))

	p, err := purser.New(h, kr)
	assert.Ok(tb, err, "pursertest: purser.New")

	return p, lck
}

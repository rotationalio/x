package benchmark_test

// Fixtures and helpers shared by benchmark bodies and Benchmark* entry points.

import (
	"context"
	"fmt"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/purser/keyring/memring"
	"go.rtnl.ai/x/purser/locker"
	lockerv1 "go.rtnl.ai/x/purser/locker/v1"
)

const benchNS = "bench-ns"

// benchSeed is a fixed 32-byte X25519 seed (deterministic locker identity).
var benchSeed = [32]byte{
	0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
	0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
	0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
}

// benchSizes are plaintext payload sizes exercised by sub-benchmarks.
var benchSizes = []int{16, 64, 256, 4096}

// benchCtx is shared by Purser benchmark loops.
var benchCtx = context.Background()

// newV1Locker returns a v1 locker from the fixed bench seed.
func newV1Locker(b *testing.B) locker.Locker {
	b.Helper()
	lck, err := lockerv1.FromSeed(benchSeed[:])
	assert.Ok(b, err, "FromSeed")
	return lck
}

// sealWire seals a corpus-backed plaintext of size bytes and returns wire.
func sealWire(b *testing.B, lck locker.Locker, size int) []byte {
	b.Helper()
	scratch := make([]byte, size)
	plain := nextPlain(scratch, size, 0)
	wire, err := lck.Seal(benchNS, plain)
	assert.Ok(b, err, "Seal setup")
	return wire
}

// newBenchMemring returns memring with the fixed v1 locker as default.
func newBenchMemring(b *testing.B) *memring.Memring {
	b.Helper()
	lck := newV1Locker(b)
	kr := memring.New()
	err := kr.SetDefault(lck)
	assert.Ok(b, err, "SetDefault")
	return kr
}

// sealPlain returns plaintext of size from the corpus (setup helper).
func sealPlain(b *testing.B, size int) []byte {
	b.Helper()
	scratch := make([]byte, size)
	return nextPlain(scratch, size, 0)
}

// sizeLabel formats a sub-benchmark size segment.
func sizeLabel(size int) string {
	return fmt.Sprintf("size=%d", size)
}

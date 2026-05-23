/*
Package benchmark_test provides hot-path benchmarks for purser (locker, keyring,
registry, and row operations). Fixtures use a fixed v1 seed, memhold, memring,
and a rotating hex plaintext corpus (plaintext.txt; see genplaintext.sh).

Run: go test -run=^$ -bench=. -benchmem ./purser/benchmark
*/
package benchmark_test

// Shared fixtures and helpers for hot-path benchmarks.

import (
	"context"
	"fmt"
	"testing"

	"go.rtnl.ai/x/purser/contract"
	"go.rtnl.ai/x/purser/keyring/memring"
	lockerv1 "go.rtnl.ai/x/purser/locker/v1"

	_ "go.rtnl.ai/x/purser" // register locker v1 via install.go
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

var benchCtx = context.Background()

// newV1Locker returns a v1 locker from the fixed bench seed.
func newV1Locker(b *testing.B) contract.Locker {
	b.Helper()
	lck, err := lockerv1.FromSeed(benchSeed[:])
	if err != nil {
		b.Fatalf("FromSeed: %v", err)
	}
	return lck
}

// sealWire seals a corpus-backed plaintext of size bytes and returns wire.
func sealWire(b *testing.B, lck contract.Locker, size int) []byte {
	b.Helper()
	scratch := make([]byte, size)
	plain := nextPlain(scratch, size, 0)
	wire, err := lck.Seal(benchNS, plain)
	if err != nil {
		b.Fatalf("Seal setup: %v", err)
	}
	return wire
}

// newBenchMemring returns memring with the fixed v1 locker active.
func newBenchMemring(b *testing.B) *memring.Memring {
	b.Helper()
	lck := newV1Locker(b)
	kr, err := memring.New(lck)
	if err != nil {
		b.Fatalf("memring.New: %v", err)
	}
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

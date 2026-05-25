/*
Package benchmark_test provides hot-path benchmarks for purser (locker, keyring,
registry, and row operations). Fixtures use a fixed v1 seed, memhold, memring,
and a rotating hex plaintext corpus (plaintext.txt; see genplaintext.sh).

Run: go test -run=^$ -bench=. -benchmem ./purser/benchmark
Optional JSON snapshot (~1s): PURSER_BENCH_SNAPSHOT=1 go test -run=TestBenchmarkSnapshot -count=1 ./purser/benchmark
Compare captures: python3 ./purser/benchmark/compare.py
*/
package benchmark_test

// Shared benchmark bodies invoked from Benchmark* entry points.

import (
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/purser"
	"go.rtnl.ai/x/purser/hold"
	hexid "go.rtnl.ai/x/purser/hold/identifier/hex"
	"go.rtnl.ai/x/purser/keyring/registry"
	"go.rtnl.ai/x/purser/pursertest"
)

// benchLockerSeal runs v1 Seal at the given plaintext size.
func benchLockerSeal(b *testing.B, size int) {
	lck := newV1Locker(b)
	scratch := make([]byte, size)
	b.SetBytes(int64(size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plain := nextPlain(scratch, size, i)
		_, err := lck.Seal(benchNS, plain)
		assert.Ok(b, err)
	}
}

// benchLockerOpen runs v1 Open at the given wire size (wire prebuilt outside timer).
func benchLockerOpen(b *testing.B, size int) {
	lck := newV1Locker(b)
	wire := sealWire(b, lck, size)
	b.SetBytes(int64(size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := lck.Open(benchNS, wire)
		assert.Ok(b, err)
	}
}

// benchLockerRoundTrip runs Seal then Open per iteration at size.
func benchLockerRoundTrip(b *testing.B, size int) {
	lck := newV1Locker(b)
	scratch := make([]byte, size)
	b.SetBytes(int64(size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plain := nextPlain(scratch, size, i)
		wire, err := lck.Seal(benchNS, plain)
		assert.Ok(b, err)
		_, err = lck.Open(benchNS, wire)
		assert.Ok(b, err)
	}
}

// benchLockerParseKeyID runs ParseKeyID on fixed wire at size.
func benchLockerParseKeyID(b *testing.B, size int) {
	lck := newV1Locker(b)
	wire := sealWire(b, lck, size)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := lck.ParseKeyID(wire)
		assert.Ok(b, err)
	}
}

// benchKeyringRoute runs memring Route on v1 wire at size.
func benchKeyringRoute(b *testing.B, size int) {
	kr := newBenchMemring(b)
	wire := sealWire(b, newV1Locker(b), size)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := kr.Route(wire)
		assert.Ok(b, err)
	}
}

// benchKeyringDefaultSeal runs LockerFor().Seal via memring at size.
func benchKeyringDefaultSeal(b *testing.B, size int) {
	kr := newBenchMemring(b)
	scratch := make([]byte, size)
	b.SetBytes(int64(size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plain := nextPlain(scratch, size, i)
		lck, err := kr.LockerFor(benchNS)
		assert.Ok(b, err)
		_, err = lck.Seal(benchNS, plain)
		assert.Ok(b, err)
	}
}

// benchRegistryParseKeyID runs registry.ParseKeyID on v1 wire at size.
func benchRegistryParseKeyID(b *testing.B, size int) {
	lck := newV1Locker(b)
	wire := sealWire(b, lck, size)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := registry.ParseKeyID(wire)
		assert.Ok(b, err)
	}
}

// benchPurserStore runs Purser.Store at size (reuses one purser for the timed loop).
func benchPurserStore(b *testing.B, size int) {
	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(b, err)
	kr := newBenchMemring(b)
	p, err := purser.New(h, kr)
	assert.Ok(b, err)
	scratch := make([]byte, size)
	b.SetBytes(int64(size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plain := nextPlain(scratch, size, i)
		_, err := p.Store(benchCtx, benchNS, plain)
		assert.Ok(b, err)
	}
}

// benchPurserRetrieve runs Purser.Retrieve on a pre-stored row at size.
func benchPurserRetrieve(b *testing.B, size int) {
	p, id := newBenchPurserWithSize(b, size)
	b.SetBytes(int64(size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Retrieve(benchCtx, benchNS, id)
		assert.Ok(b, err)
	}
}

// benchPurserNulllockerStore runs pursertest Store without v1 crypto at size.
func benchPurserNulllockerStore(b *testing.B, size int) {
	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(b, err)
	p := pursertest.NewTestPurser(b, h)
	scratch := make([]byte, size)
	b.SetBytes(int64(size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plain := nextPlain(scratch, size, i)
		_, err := p.Store(benchCtx, benchNS, plain)
		assert.Ok(b, err)
	}
}

// benchSize256 is the payload size for the nulllocker orchestration sub-benchmark.
const benchSize256 = 256

// newBenchPurserWithSize stores one row at size and returns purser + id.
func newBenchPurserWithSize(b *testing.B, size int) (*purser.Purser, string) {
	b.Helper()
	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(b, err, "NewMemHold")
	kr := newBenchMemring(b)
	p, err := purser.New(h, kr)
	assert.Ok(b, err, "purser.New")
	res, err := p.Store(benchCtx, benchNS, sealPlain(b, size))
	assert.Ok(b, err, "Store setup")
	return p, res.ID
}

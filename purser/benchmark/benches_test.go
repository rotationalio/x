package benchmark_test

// Shared benchmark bodies invoked from Benchmark* entry points.

import (
	"testing"

	"go.rtnl.ai/x/purser"
	"go.rtnl.ai/x/purser/contract"
	"go.rtnl.ai/x/purser/hold"
	hexid "go.rtnl.ai/x/purser/hold/identifier/hex"
	"go.rtnl.ai/x/purser/pursertest"
	"go.rtnl.ai/x/purser/registry"
)

const benchSize256 = 256

// benchLockerSeal runs v1 Seal at the given plaintext size.
func benchLockerSeal(b *testing.B, size int) {
	lck := newV1Locker(b)
	scratch := make([]byte, size)
	b.SetBytes(int64(size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plain := nextPlain(scratch, size, i)
		if _, err := lck.Seal(benchNS, plain); err != nil {
			b.Fatal(err)
		}
	}
}

// benchLockerOpen runs v1 Open at the given wire size (wire prebuilt outside timer).
func benchLockerOpen(b *testing.B, size int) {
	lck := newV1Locker(b)
	wire := sealWire(b, lck, size)
	b.SetBytes(int64(size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := lck.Open(benchNS, wire); err != nil {
			b.Fatal(err)
		}
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
		if err != nil {
			b.Fatal(err)
		}
		if _, err = lck.Open(benchNS, wire); err != nil {
			b.Fatal(err)
		}
	}
}

// benchLockerParseKeyID runs ParseKeyID on fixed wire at size.
func benchLockerParseKeyID(b *testing.B, size int) {
	lck := newV1Locker(b)
	wire := sealWire(b, lck, size)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := lck.ParseKeyID(wire); err != nil {
			b.Fatal(err)
		}
	}
}

// benchKeyringRouteKeyID runs memring RouteKeyID on v1 wire at size.
func benchKeyringRouteKeyID(b *testing.B, size int) {
	kr := newBenchMemring(b)
	wire := sealWire(b, kr.Active(), size)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := kr.RouteKeyID(wire); err != nil {
			b.Fatal(err)
		}
	}
}

// benchKeyringActiveSeal runs Active().Seal via memring at size.
func benchKeyringActiveSeal(b *testing.B, size int) {
	kr := newBenchMemring(b)
	scratch := make([]byte, size)
	b.SetBytes(int64(size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plain := nextPlain(scratch, size, i)
		if _, err := kr.Active().Seal(benchNS, plain); err != nil {
			b.Fatal(err)
		}
	}
}

// benchRegistryParseKeyID runs registry.ParseKeyID on v1 wire at size.
func benchRegistryParseKeyID(b *testing.B, size int) {
	lck := newV1Locker(b)
	wire := sealWire(b, lck, size)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := registry.ParseKeyID(wire); err != nil {
			b.Fatal(err)
		}
	}
}

// benchPurserStore runs Purser.Store at size (reuses one purser for the timed loop).
func benchPurserStore(b *testing.B, size int) {
	h, err := hold.NewMemHold(hexid.Identifier{})
	if err != nil {
		b.Fatal(err)
	}
	kr := newBenchMemring(b)
	p, err := purser.New(h, kr)
	if err != nil {
		b.Fatal(err)
	}
	scratch := make([]byte, size)
	b.SetBytes(int64(size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plain := nextPlain(scratch, size, i)
		if _, err := p.Store(benchCtx, benchNS, plain); err != nil {
			b.Fatal(err)
		}
	}
}

// benchPurserRetrieve runs Purser.Retrieve on a pre-stored row at size.
func benchPurserRetrieve(b *testing.B, size int) {
	p, id := newBenchPurserWithSize(b, size)
	b.SetBytes(int64(size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := p.Retrieve(benchCtx, benchNS, id); err != nil {
			b.Fatal(err)
		}
	}
}

// benchPurserNulllockerStore runs pursertest Store without v1 crypto at size.
func benchPurserNulllockerStore(b *testing.B, size int) {
	h, err := hold.NewMemHold(hexid.Identifier{})
	if err != nil {
		b.Fatal(err)
	}
	p := pursertest.NewTestPurser(b, h)
	scratch := make([]byte, size)
	b.SetBytes(int64(size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plain := nextPlain(scratch, size, i)
		if _, err := p.Store(benchCtx, benchNS, plain); err != nil {
			b.Fatal(err)
		}
	}
}

// newBenchPurserWithSize stores one row at size and returns purser + id.
func newBenchPurserWithSize(b *testing.B, size int) (contract.Purser, string) {
	b.Helper()
	h, err := hold.NewMemHold(hexid.Identifier{})
	if err != nil {
		b.Fatalf("NewMemHold: %v", err)
	}
	kr := newBenchMemring(b)
	p, err := purser.New(h, kr)
	if err != nil {
		b.Fatalf("purser.New: %v", err)
	}
	id, err := p.Store(benchCtx, benchNS, sealPlain(b, size))
	if err != nil {
		b.Fatalf("Store setup: %v", err)
	}
	return p, id
}

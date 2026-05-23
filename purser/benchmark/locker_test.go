package benchmark_test

// BenchmarkLocker measures v1 locker Seal, Open, RoundTrip, and ParseKeyID.

import "testing"

// BenchmarkLocker runs hot-path v1 locker crypto benchmarks at several plaintext sizes.
func BenchmarkLocker(b *testing.B) {
	for _, size := range benchSizes {
		b.Run("Seal/"+sizeLabel(size), func(b *testing.B) {
			benchLockerSeal(b, size)
		})
		b.Run("Open/"+sizeLabel(size), func(b *testing.B) {
			benchLockerOpen(b, size)
		})
		b.Run("RoundTrip/"+sizeLabel(size), func(b *testing.B) {
			benchLockerRoundTrip(b, size)
		})
		b.Run("ParseKeyID/"+sizeLabel(size), func(b *testing.B) {
			benchLockerParseKeyID(b, size)
		})
	}
}

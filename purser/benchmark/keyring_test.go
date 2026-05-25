package benchmark_test

// BenchmarkKeyring measures memring Route and default-locker Seal.

import "testing"

// BenchmarkKeyring runs in-memory keyring routing and default-locker seal benchmarks.
func BenchmarkKeyring(b *testing.B) {
	for _, size := range benchSizes {
		b.Run("Route/1Locker/"+sizeLabel(size), func(b *testing.B) {
			benchKeyringRoute(b, size)
		})
		b.Run("DefaultSeal/"+sizeLabel(size), func(b *testing.B) {
			benchKeyringDefaultSeal(b, size)
		})
	}
}

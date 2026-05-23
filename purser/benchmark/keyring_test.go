package benchmark_test

// BenchmarkKeyring measures memring RouteKeyID and Active().Seal.

import "testing"

// BenchmarkKeyring runs in-memory keyring routing and active-locker seal benchmarks.
func BenchmarkKeyring(b *testing.B) {
	for _, size := range benchSizes {
		b.Run("RouteKeyID/1Locker/"+sizeLabel(size), func(b *testing.B) {
			benchKeyringRouteKeyID(b, size)
		})
		b.Run("ActiveSeal/"+sizeLabel(size), func(b *testing.B) {
			benchKeyringActiveSeal(b, size)
		})
	}
}

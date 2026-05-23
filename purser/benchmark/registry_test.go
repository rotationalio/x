package benchmark_test

// BenchmarkRegistry measures registry.ParseKeyID (metadata routing, not row decrypt).

import "testing"

// BenchmarkRegistry runs registry.ParseKeyID at several wire sizes.
func BenchmarkRegistry(b *testing.B) {
	for _, size := range benchSizes {
		b.Run("ParseKeyID/"+sizeLabel(size), func(b *testing.B) {
			benchRegistryParseKeyID(b, size)
		})
	}
}

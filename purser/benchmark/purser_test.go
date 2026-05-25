package benchmark_test

// BenchmarkPurser measures end-to-end Store and Retrieve on memhold + v1 memring.

import "testing"

// BenchmarkPurser runs orchestration benchmarks for the full Purser row path.
func BenchmarkPurser(b *testing.B) {
	for _, size := range benchSizes {
		b.Run("Store/"+sizeLabel(size), func(b *testing.B) {
			benchPurserStore(b, size)
		})
		b.Run("Retrieve/"+sizeLabel(size), func(b *testing.B) {
			benchPurserRetrieve(b, size)
		})
	}
	b.Run("Orchestration/Nulllocker/Store/"+sizeLabel(benchSize256), func(b *testing.B) {
		benchPurserNulllockerStore(b, benchSize256)
	})
}

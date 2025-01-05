package randstr_test

import (
	"math/rand"
	"regexp"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/randstr"
)

func TestAlpha(t *testing.T) {
	// This is a long running test, skip if in short mode
	if testing.Short() {
		t.Skip("skipping long running test in short mode")
	}

	// Test creating different random strings at different lengths
	for i := 0; i < 10000; i++ {
		len := rand.Intn(512) + 1
		alpha := randstr.Alpha(len)
		assert.Len(t, alpha, len)
		assert.Regexp(t, regexp.MustCompile(`^[a-zA-Z]+$`), alpha)
	}

	vals := make(map[string]struct{})
	for i := 0; i < 10000; i++ {
		val := randstr.Alpha(16)
		vals[val] = struct{}{}
	}
	assert.Len(t, vals, 10000, "there is a very low chance that a duplicate value was generated")
}

func TestAlphaNumeric(t *testing.T) {
	// This is a long running test, skip if in short mode
	if testing.Short() {
		t.Skip("skipping long running test in short mode")
	}

	// Test creating different random strings at different lengths
	for i := 0; i < 10000; i++ {
		len := rand.Intn(512) + 1
		alpha := randstr.AlphaNumeric(len)
		assert.Len(t, alpha, len)
		assert.Regexp(t, regexp.MustCompile(`^[a-zA-Z0-9]+$`), alpha)
	}

	vals := make(map[string]struct{})
	for i := 0; i < 10000; i++ {
		val := randstr.AlphaNumeric(16)
		vals[val] = struct{}{}
	}
	assert.Len(t, vals, 10000, "there is a very low chance that a duplicate value was generated")
}

func TestPassword(t *testing.T) {
	// This is a long running test, skip if in short mode
	if testing.Short() {
		t.Skip("skipping long running test in short mode")
	}

	// Test creating different random strings at different lengths
	for i := 0; i < 10000; i++ {
		len := rand.Intn(512) + 1
		alpha := randstr.Password(len)
		assert.Len(t, alpha, len)
		assert.Regexp(t, regexp.MustCompile(`^[a-zA-Z0-9\!@#\$%\^&*()_\+\-=\[\]\{\};:\\|,.<>/\?;]+$`), alpha)
	}

	vals := make(map[string]struct{})
	for i := 0; i < 10000; i++ {
		val := randstr.Password(16)
		vals[val] = struct{}{}
	}
	assert.Len(t, vals, 10000, "there is a very low chance that a duplicate value was generated")
}

func TestWord(t *testing.T) {
	assert.Equal(t, "", randstr.Word(0), "empty string with empty word len")

	// This is a long running test, skip if in short mode
	if testing.Short() {
		t.Skip("skipping long running test in short mode")
	}

	// Test creating different random strings at different lengths
	for i := 0; i < 10000; i++ {
		len := rand.Intn(512) + 1
		alpha := randstr.Word(len)
		assert.Len(t, alpha, len)
		assert.Regexp(t, regexp.MustCompile(`^[bcdfghjklmnpqrstvwxzaeiou]+$`), alpha)
	}

	vals := make(map[string]struct{})
	for i := 0; i < 10000; i++ {
		val := randstr.Word(16)
		vals[val] = struct{}{}
	}
	assert.Len(t, vals, 10000, "there is a very low chance that a duplicate value was generated")
}

func TestCryptoRandInt(t *testing.T) {
	// This is a long running test, skip if in short mode
	if testing.Short() {
		t.Skip("skipping long running test in short mode")
	}

	nums := make(map[uint64]struct{})
	for i := 0; i < 10000; i++ {
		val := randstr.CryptoRandInt()
		nums[val] = struct{}{}
	}
	assert.Len(t, nums, 10000, "there is a very low chance that a duplicate value was generated")
}

func benchmarkAlpha(i int, b *testing.B) {
	for n := 0; n < b.N; n++ {
		randstr.Alpha(i)
	}
}

func benchmarkAlphaNumeric(i int, b *testing.B) {
	for n := 0; n < b.N; n++ {
		randstr.AlphaNumeric(i)
	}
}

func BenchmarkAlpha16(b *testing.B)  { benchmarkAlpha(16, b) }
func BenchmarkAlpha64(b *testing.B)  { benchmarkAlpha(64, b) }
func BenchmarkAlpha256(b *testing.B) { benchmarkAlpha(256, b) }

func BenchmarkAlphaNumeric16(b *testing.B)  { benchmarkAlphaNumeric(16, b) }
func BenchmarkAlphaNumeric64(b *testing.B)  { benchmarkAlphaNumeric(64, b) }
func BenchmarkAlphaNumeric256(b *testing.B) { benchmarkAlphaNumeric(256, b) }

func BenchmarkCryptoRandInt(b *testing.B) {
	for n := 0; n < b.N; n++ {
		randstr.CryptoRandInt()
	}
}

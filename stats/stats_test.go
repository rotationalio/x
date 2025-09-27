package stats_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/stats"
)

var (
	dataMu        sync.Once
	testFloats    []float64
	testInts      []int64
	testDurations []time.Duration
)

const (
	testSeed = 512     // random seed ensures that we'll always get the same test data
	testN    = 1000000 // number of samples to generate
	delta    = 1e-9    // delta for comparison tests
)

func loadTestData() {
	dataMu.Do(func() {
		source := rand.NewSource(testSeed)
		r := rand.New(source)

		testFloats = make([]float64, testN)
		testInts = make([]int64, testN)
		testDurations = make([]time.Duration, testN)

		for i := 0; i < testN; i++ {
			testFloats[i] = r.NormFloat64()
			testInts[i] = r.Int63n(2400) + 320
			testDurations[i] = time.Duration(testFloats[i]*103.5+1200.00) * time.Millisecond
		}
	})
}

func ExampleStatistics() {
	// Test setup
	loadTestData()
	samples := testFloats

	// Example usage with float64 samples starts here.
	stats := new(stats.Statistics[float64])
	for _, sample := range samples {
		stats.Update(sample)
	}

	data, _ := json.MarshalIndent(stats, "", "  ")
	fmt.Println(string(data))
	// Output:
	// {
	//   "maximum": 4.671499986908815,
	//   "mean": -0.0004712672972730699,
	//   "minimum": -4.962035104188279,
	//   "range": 9.633535091097094,
	//   "samples": 1000000,
	//   "squares": 999823.5614339677,
	//   "stddev": 0.9999121657252907,
	//   "total": -471.2672972730699,
	//   "variance": 0.9998243391654413
	// }
}

func TestStatistics(t *testing.T) {
	loadTestData()

	t.Run("float64", func(t *testing.T) {
		stats := new(stats.Statistics[float64])
		stats.Update(testFloats...)

		assert.Equal(t, int64(1000000), stats.N())
		assert.InDelta(t, -471.2672972730699, stats.Total(), delta)
		assert.InDelta(t, -0.0004712672972730699, stats.Mean(), delta)
		assert.InDelta(t, 0.9999121657252907, stats.StdDev(), delta)
		assert.InDelta(t, 0.9998243391654413, stats.Variance(), delta)
		assert.InDelta(t, 4.671499986908815, stats.Maximum(), delta)
		assert.InDelta(t, -4.962035104188279, stats.Minimum(), delta)
		assert.InDelta(t, 9.633535091097094, float64(stats.Range()), delta)
	})

	t.Run("int64", func(t *testing.T) {
		stats := new(stats.Statistics[int64])
		stats.Update(testInts...)

		assert.Equal(t, int64(1000000), stats.N())
		assert.InDelta(t, 1519201574, stats.Total(), delta)
		assert.InDelta(t, 1519.201574, stats.Mean(), delta)
		assert.InDelta(t, 692.8467630569978, stats.StdDev(), delta)
		assert.InDelta(t, 480036.6370785597, stats.Variance(), delta)
		assert.InDelta(t, 2719, stats.Maximum(), delta)
		assert.InDelta(t, 320, stats.Minimum(), delta)
		assert.InDelta(t, 2399, float64(stats.Range()), delta)
	})

	t.Run("time.Duration", func(t *testing.T) {
		stats := new(stats.Statistics[time.Duration])
		stats.Update(testDurations...)

		t.Log(strconv.FormatInt(int64(stats.Range()), 10))

		assert.Equal(t, int64(1000000), stats.N())
		assert.InDelta(t, time.Duration(1199451546000000), stats.Total(), 1e-3)
		assert.InDelta(t, 1.199451546e+09, stats.Mean(), 1e-3)
		assert.InDelta(t, 1.0349137617525254e+08, stats.StdDev(), 1e-3)
		assert.InDelta(t, 1.071046494264763e+16, stats.Variance(), 1e-3)
		assert.InDelta(t, time.Duration(1683000000), stats.Maximum(), 1e-3)
		assert.InDelta(t, time.Duration(686000000), stats.Minimum(), 1e-3)
		assert.InDelta(t, time.Duration(997000000), float64(stats.Range()), 1e-3)
	})
}

func TestAppend(t *testing.T) {
	loadTestData()

	t.Run("Combine", func(t *testing.T) {
		src := new(stats.Statistics[float64])
		src.Update(testFloats[:len(testFloats)/2]...)

		dst := new(stats.Statistics[float64])
		dst.Update(testFloats[len(testFloats)/2:]...)

		// Test values before append
		assert.Equal(t, int64(500000), dst.N())
		assert.InDelta(t, 174.87440039993456, dst.Total(), delta)
		assert.InDelta(t, 0.0003497488007998691, dst.Mean(), delta)
		assert.InDelta(t, 1.0002589687226628, dst.StdDev(), delta)
		assert.InDelta(t, 1.0005180045101252, dst.Variance(), delta)
		assert.InDelta(t, 4.3401303694346405, dst.Maximum(), delta)
		assert.InDelta(t, -4.962035104188279, dst.Minimum(), delta)
		assert.InDelta(t, 9.30216547362292, float64(dst.Range()), delta)

		assert.Equal(t, int64(500000), src.N())
		assert.InDelta(t, -646.1416976730367, src.Total(), delta)
		assert.InDelta(t, -0.0012922833953460733, src.Mean(), delta)
		assert.InDelta(t, 0.9995655683024451, src.StdDev(), delta)
		assert.InDelta(t, 0.9991313253357902, src.Variance(), delta)
		assert.InDelta(t, 4.671499986908815, src.Maximum(), delta)
		assert.InDelta(t, -4.285186672658051, src.Minimum(), delta)
		assert.InDelta(t, 8.956686659566866, float64(src.Range()), delta)

		// Append src into dst
		dst.Append(src)

		// Test values after append (values should have changed)
		assert.Equal(t, int64(1000000), dst.N())
		assert.InDelta(t, -471.26729727310214, dst.Total(), delta)
		assert.InDelta(t, -0.0004712672972730699, dst.Mean(), delta)
		assert.InDelta(t, 0.9999121657252703, dst.StdDev(), delta)
		assert.InDelta(t, 0.9998243391654004, dst.Variance(), delta)
		assert.InDelta(t, 4.671499986908815, dst.Maximum(), delta)
		assert.InDelta(t, -4.962035104188279, dst.Minimum(), delta)
		assert.InDelta(t, 9.633535091097094, float64(dst.Range()), delta)

		// Ensure src is unchanged
		assert.Equal(t, int64(500000), src.N())
		assert.InDelta(t, -646.1416976730367, src.Total(), delta)
		assert.InDelta(t, -0.0012922833953460733, src.Mean(), delta)
		assert.InDelta(t, 0.9995655683024451, src.StdDev(), delta)
		assert.InDelta(t, 0.9991313253357902, src.Variance(), delta)
		assert.InDelta(t, 4.671499986908815, src.Maximum(), delta)
		assert.InDelta(t, -4.285186672658051, src.Minimum(), delta)
		assert.InDelta(t, 8.956686659566866, float64(src.Range()), delta)

	})

	t.Run("Empty", func(t *testing.T) {
		t.Run("Src", func(t *testing.T) {
			src := new(stats.Statistics[float64])
			dst := new(stats.Statistics[float64])
			dst.Update(testFloats[len(testFloats)/2:]...)

			// Test values before append
			assert.Equal(t, int64(500000), dst.N())
			assert.InDelta(t, 174.87440039993456, dst.Total(), delta)
			assert.InDelta(t, 0.0003497488007998691, dst.Mean(), delta)
			assert.InDelta(t, 1.0002589687226628, dst.StdDev(), delta)
			assert.InDelta(t, 1.0005180045101252, dst.Variance(), delta)
			assert.InDelta(t, 4.3401303694346405, dst.Maximum(), delta)
			assert.InDelta(t, -4.962035104188279, dst.Minimum(), delta)
			assert.InDelta(t, 9.30216547362292, float64(dst.Range()), delta)

			assert.Equal(t, int64(0), src.N())
			assert.InDelta(t, 0, src.Total(), delta)
			assert.InDelta(t, 0, src.Mean(), delta)
			assert.InDelta(t, 0, src.StdDev(), delta)
			assert.InDelta(t, 0, src.Variance(), delta)
			assert.InDelta(t, 0, src.Maximum(), delta)
			assert.InDelta(t, 0, src.Minimum(), delta)
			assert.InDelta(t, 0, float64(src.Range()), delta)

			// Append src into dst
			dst.Append(src)

			// Test values after append (values should be unchanged)
			assert.Equal(t, int64(500000), dst.N())
			assert.InDelta(t, 174.87440039993456, dst.Total(), delta)
			assert.InDelta(t, 0.0003497488007998691, dst.Mean(), delta)
			assert.InDelta(t, 1.0002589687226628, dst.StdDev(), delta)
			assert.InDelta(t, 1.0005180045101252, dst.Variance(), delta)
			assert.InDelta(t, 4.3401303694346405, dst.Maximum(), delta)
			assert.InDelta(t, -4.962035104188279, dst.Minimum(), delta)
			assert.InDelta(t, 9.30216547362292, float64(dst.Range()), delta)

			assert.Equal(t, int64(0), src.N())
			assert.InDelta(t, 0, src.Total(), delta)
			assert.InDelta(t, 0, src.Mean(), delta)
			assert.InDelta(t, 0, src.StdDev(), delta)
			assert.InDelta(t, 0, src.Variance(), delta)
			assert.InDelta(t, 0, src.Maximum(), delta)
			assert.InDelta(t, 0, src.Minimum(), delta)
			assert.InDelta(t, 0, float64(src.Range()), delta)
		})

		t.Run("Dst", func(t *testing.T) {
			src := new(stats.Statistics[float64])
			dst := new(stats.Statistics[float64])
			src.Update(testFloats[len(testFloats)/2:]...)

			// Test values before append
			assert.Equal(t, int64(0), dst.N())
			assert.InDelta(t, 0, dst.Total(), delta)
			assert.InDelta(t, 0, dst.Mean(), delta)
			assert.InDelta(t, 0, dst.StdDev(), delta)
			assert.InDelta(t, 0, dst.Variance(), delta)
			assert.InDelta(t, 0, dst.Maximum(), delta)
			assert.InDelta(t, 0, dst.Minimum(), delta)
			assert.InDelta(t, 0, float64(dst.Range()), delta)

			assert.Equal(t, int64(500000), src.N())
			assert.InDelta(t, 174.87440039993456, src.Total(), delta)
			assert.InDelta(t, 0.0003497488007998691, src.Mean(), delta)
			assert.InDelta(t, 1.0002589687226628, src.StdDev(), delta)
			assert.InDelta(t, 1.0005180045101252, src.Variance(), delta)
			assert.InDelta(t, 4.3401303694346405, src.Maximum(), delta)
			assert.InDelta(t, -4.962035104188279, src.Minimum(), delta)
			assert.InDelta(t, 9.30216547362292, float64(src.Range()), delta)

			// Append src into dst
			dst.Append(src)

			// Test values after append
			assert.Equal(t, int64(500000), dst.N())
			assert.InDelta(t, 174.87440039993456, dst.Total(), delta)
			assert.InDelta(t, 0.0003497488007998691, dst.Mean(), delta)
			assert.InDelta(t, 1.0002589687226628, dst.StdDev(), delta)
			assert.InDelta(t, 1.0005180045101252, dst.Variance(), delta)
			assert.InDelta(t, 4.3401303694346405, dst.Maximum(), delta)
			assert.InDelta(t, -4.962035104188279, dst.Minimum(), delta)
			assert.InDelta(t, 9.30216547362292, float64(dst.Range()), delta)

			// Ensure src is unchanged
			assert.Equal(t, int64(500000), src.N())
			assert.InDelta(t, 174.87440039993456, src.Total(), delta)
			assert.InDelta(t, 0.0003497488007998691, src.Mean(), delta)
			assert.InDelta(t, 1.0002589687226628, src.StdDev(), delta)
			assert.InDelta(t, 1.0005180045101252, src.Variance(), delta)
			assert.InDelta(t, 4.3401303694346405, src.Maximum(), delta)
			assert.InDelta(t, -4.962035104188279, src.Minimum(), delta)
			assert.InDelta(t, 9.30216547362292, float64(src.Range()), delta)
		})
	})

}

func TestJSON(t *testing.T) {
	loadTestData()

	orig := new(stats.Statistics[float64])
	orig.Update(testFloats...)

	data, err := json.Marshal(orig)
	assert.Ok(t, err)

	cmpt := new(stats.Statistics[float64])
	err = json.Unmarshal(data, cmpt)
	assert.Ok(t, err)

	assert.Equal(t, orig.N(), cmpt.N())
	assert.InDelta(t, orig.Total(), cmpt.Total(), delta)
	assert.InDelta(t, orig.Mean(), cmpt.Mean(), delta)
	assert.InDelta(t, orig.StdDev(), cmpt.StdDev(), delta)
	assert.InDelta(t, orig.Variance(), cmpt.Variance(), delta)
	assert.InDelta(t, float64(orig.Maximum()), float64(cmpt.Maximum()), delta)
	assert.InDelta(t, float64(orig.Minimum()), float64(cmpt.Minimum()), delta)
	assert.InDelta(t, float64(orig.Range()), float64(cmpt.Range()), delta)
}

func TestBadJSON(t *testing.T) {
	testCases := []struct {
		in       string
		obj      any
		expected string
	}{
		{
			in:       `{"foo":"NaN"}`,
			obj:      new(stats.Statistics[float64]),
			expected: `json: invalid number literal, trying to unmarshal "\"NaN\"" into Number`,
		},
		{
			in:       `{"total": 9.813435956500003, "squares": 103.123456789, "maximum": 15.45832771, "minimum": 5.51224787}`,
			obj:      new(stats.Statistics[float64]),
			expected: `missing samples field`,
		},
		{
			in:       `{"samples": 1000, "squares": 103.123456789, "maximum": 15.45832771, "minimum": 5.51224787}`,
			obj:      new(stats.Statistics[float64]),
			expected: `missing total field`,
		},
		{
			in:       `{"samples": 1000, "total": 9.813435956500003, "maximum": 15.45832771, "minimum": 5.51224787}`,
			obj:      new(stats.Statistics[float64]),
			expected: `missing squares field`,
		},
		{
			in:       `{"samples": 1000, "total": 9.813435956500003, "squares": 103.123456789, "minimum": 5.51224787}`,
			obj:      new(stats.Statistics[float64]),
			expected: `missing maximum field`,
		},
		{
			in:       `{"samples": 1000, "total": 9.813435956500003, "squares": 103.123456789, "maximum": 15.45832771}`,
			obj:      new(stats.Statistics[float64]),
			expected: `missing minimum field`,
		},
	}

	for i, tc := range testCases {
		err := json.Unmarshal([]byte(tc.in), &tc.obj)
		assert.EqualError(t, err, tc.expected, "expected an error on test case %d", i)
	}
}

func TestBinary(t *testing.T) {
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	orig := new(stats.Statistics[float64])

	for i := 0; i < 10000; i++ {
		orig.Update(rand.NormFloat64()*4.21312 + 15.930541)
	}

	data, err := orig.MarshalBinary()
	assert.Ok(t, err)

	cmpt := new(stats.Statistics[float64])
	err = cmpt.UnmarshalBinary(data)
	assert.Ok(t, err)

	assert.Equal(t, orig.N(), cmpt.N())
	assert.InDelta(t, orig.Total(), cmpt.Total(), delta)
	assert.InDelta(t, orig.Mean(), cmpt.Mean(), delta)
	assert.InDelta(t, orig.StdDev(), cmpt.StdDev(), delta)
	assert.InDelta(t, orig.Variance(), cmpt.Variance(), delta)
	assert.InDelta(t, float64(orig.Maximum()), float64(cmpt.Maximum()), delta)
	assert.InDelta(t, float64(orig.Minimum()), float64(cmpt.Minimum()), delta)
	assert.InDelta(t, float64(orig.Range()), float64(cmpt.Range()), delta)

}

func TestString(t *testing.T) {
	loadTestData()
	stats := new(stats.Statistics[float64])
	stats.Update(testFloats...)

	str := stats.String()
	expected := "-0.000µ ± 1.000σ in [-4.962035104188279, 4.671499986908815] for 1000000 samples"
	assert.Equal(t, expected, str)
}

func BenchmarkStatistics_Update(b *testing.B) {
	b.Run("float64", func(b *testing.B) {
		rand42 := rand.New(rand.NewSource(42))
		stats := new(stats.Statistics[float64])
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			val := rand42.Float64()
			stats.Update(val)
		}
	})

	b.Run("int64", func(b *testing.B) {
		rand42 := rand.New(rand.NewSource(42))
		stats := new(stats.Statistics[int64])
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			val := rand42.Int63n(2400) + 320
			stats.Update(val)
		}
	})

	b.Run("time.Duration", func(b *testing.B) {
		rand42 := rand.New(rand.NewSource(42))
		stats := new(stats.Statistics[time.Duration])
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			val := time.Duration(rand42.Float64()*103.5+1200.00) * time.Millisecond
			stats.Update(val)
		}
	})
}

func BenchmarkStatistics_Sequential(b *testing.B) {
	loadTestData()

	b.Run("float64", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			stats := new(stats.Statistics[float64])
			for _, val := range testFloats {
				stats.Update(val)
			}
		}
	})

	b.Run("int64", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			stats := new(stats.Statistics[int64])
			for _, val := range testInts {
				stats.Update(val)
			}
		}
	})

	b.Run("time.Duration", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			stats := new(stats.Statistics[time.Duration])
			for _, val := range testDurations {
				stats.Update(val)
			}
		}
	})
}

func BenchmarkStatistics_BulkLoad(b *testing.B) {
	loadTestData()

	b.Run("float64", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			stats := new(stats.Statistics[float64])
			stats.Update(testFloats...)
		}
	})

	b.Run("int64", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			stats := new(stats.Statistics[int64])
			stats.Update(testInts...)
		}
	})

	b.Run("time.Duration", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			stats := new(stats.Statistics[time.Duration])
			stats.Update(testDurations...)
		}
	})
}

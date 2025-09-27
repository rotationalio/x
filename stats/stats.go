/*
Package stats implements an online computation of summary statistics.

The primary idea of this package is that samples are coming in at real time,
and online computations of the shape of the distribution: the mean, variance,
and range need to be computed on-demand. Rather than keeping an array of
values, online algorithms update the internal state of the descriptive
statistics at runtime, saving memory.

To track statistics in an online fashion, you need to keep track of the
various aggregates that are used to compute the final descriptive statistics
of the distribution. For simple statistics such as the minimum, maximum,
standard deviation, and mean you need to track the number of samples, the sum
of samples, and the sum of the squares of all samples (along with the minimum
and maximum value seen).

The primary entry point into this function is the Update method, where you
can pass sample values and retrieve data back. All other methods are simply
computations for values.
*/
package stats

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
)

// Number is an interface that allows for both integer and floating-point computation.
type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// Statistics keeps track of descriptive statistics in an online fashion at
// runtime without saving each individual sample in an array. It does this by
// updating the internal state of summary aggregates including the number of
// samples seen, the sum of values, and the sum of the value squared. It also
// tracks the minimum and maximum values seen.
//
// The primary entry point to the object is via the Update method, where one
// or more samples can be passed.
//
// This implementation is not thread-safe, and should be protected by a mutex
// if being used in a concurrent context (or use ConcurrentStatistics).
type Statistics[T Number] struct {
	samples int64   // the number of samples seen
	total   float64 // the sum of all samples
	squares float64 // the sum of the squares of all samples
	maximum T       // the maximum sample seen
	minimum T       // the minimum sample seen
}

// Update the underlying statistical distribution with one or more samples.
func (s *Statistics[T]) Update(samples ...T) {
	for _, sample := range samples {
		s.samples++
		s.total += float64(sample)
		s.squares += float64(sample * sample)

		// If this is our first sample, then this value is both our maximum and minimum
		// value, otherwise perform comparisons.
		if s.samples == 1 {
			s.maximum = sample
			s.minimum = sample
		} else {
			if sample > s.maximum {
				s.maximum = sample
			}
			if sample < s.minimum {
				s.minimum = sample
			}
		}
	}
}

// N returns the number of samples observed.
func (s *Statistics[T]) N() int64 {
	return s.samples
}

// Total returns the sum of the samples.
func (s *Statistics[T]) Total() T {
	return T(s.total)
}

// Mean returns the average for all samples, computed as the sum of values
// divided by the total number of samples seen. If no samples have been added
// then this function returns 0.0. Note that 0.0 is a valid mean and does
// not necessarily mean that no samples have been tracked.
func (s *Statistics[T]) Mean() float64 {
	if s.samples > 0 {
		return s.total / float64(s.samples)
	}
	return 0.0
}

// Variance computes the variability of samples and describes the distance of
// the distribution from the mean. If one or none samples have been added to
// the data set then this function returns 0.0 (two or more values are
// required to compute variance).
func (s *Statistics[T]) Variance() float64 {
	n := float64(s.samples)
	if s.samples > 1 {
		num := (n*s.squares - s.total*s.total)
		den := (n * (n - 1))
		return num / den
	}
	return 0.0
}

// StdDev returns the standard deviation of samples, the square root of the
// variance. Two or more values are required to comput the standard deviation
// if one or none samples have been added to the data then this function
// returns 0.0.
func (s *Statistics[T]) StdDev() float64 {
	if s.samples > 1 {
		return math.Sqrt(s.Variance())
	}
	return 0.0
}

// Maximum returns the maximum value of samples seen. If no samples have been
// added to the dataset, then this function returns 0.0.
func (s *Statistics[T]) Maximum() T {
	return s.maximum
}

// Minimum returns the minimum value of samples seen. If no samples have been
// added to the dataset, then this function returns 0.0.
func (s *Statistics[T]) Minimum() T {
	return s.minimum
}

// Range returns the difference between the maximum and minimum of samples.
// If no samples have been added to the dataset, this function returns 0.0.
// This function will also return zero if the maximum value equals the
// minimum value, e.g. in the case only one sample has been added or all of
// the samples are the same value.
func (s *Statistics[T]) Range() T {
	return s.maximum - s.minimum
}

// Append another statistics object to the current statistics object,
// incrementing the distribution from the other object.
func (s *Statistics[T]) Append(o *Statistics[T]) {
	// Compute minimum and maximum aggregates by comparing both objects,
	// ensuring that zero valued items are not overriding the comparision.
	// Must come before any other aggregation.
	if o.samples > 0 {
		if o.maximum > s.maximum {
			s.maximum = o.maximum
		}

		if s.samples == 0 || o.minimum < s.minimum {
			s.minimum = o.minimum
		}
	}

	// Update the current statistics object
	s.total += o.total
	s.samples += o.samples
	s.squares += o.squares
}

func (s *Statistics[T]) MarshalJSON() ([]byte, error) {
	data := map[string]json.Number{
		"samples":  json.Number(strconv.FormatInt(s.samples, 10)),
		"total":    json.Number(strconv.FormatFloat(s.total, 'f', -1, 64)),
		"squares":  json.Number(strconv.FormatFloat(s.squares, 'f', -1, 64)),
		"maximum":  json.Number(strconv.FormatFloat(float64(s.maximum), 'f', -1, 64)),
		"minimum":  json.Number(strconv.FormatFloat(float64(s.minimum), 'f', -1, 64)),
		"mean":     json.Number(strconv.FormatFloat(s.Mean(), 'f', -1, 64)),
		"stddev":   json.Number(strconv.FormatFloat(s.StdDev(), 'f', -1, 64)),
		"variance": json.Number(strconv.FormatFloat(s.Variance(), 'f', -1, 64)),
		"range":    json.Number(strconv.FormatFloat(float64(s.Range()), 'f', -1, 64)),
	}
	return json.Marshal(data)
}

func (s *Statistics[T]) UnmarshalJSON(data []byte) (err error) {
	stats := map[string]json.Number{}
	if err := json.Unmarshal(data, &stats); err != nil {
		return err
	}

	var (
		ok  bool
		val json.Number
	)

	if val, ok = stats["samples"]; ok {
		if s.samples, err = val.Int64(); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("missing samples field")
	}

	if val, ok = stats["total"]; ok {
		if s.total, err = val.Float64(); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("missing total field")
	}

	if val, ok = stats["squares"]; ok {
		if s.squares, err = val.Float64(); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("missing squares field")
	}

	if val, ok = stats["maximum"]; ok {
		var max float64
		if max, err = val.Float64(); err != nil {
			return err
		}
		s.maximum = T(max)
	} else {
		return fmt.Errorf("missing maximum field")
	}

	if val, ok = stats["minimum"]; ok {
		var min float64
		if min, err = val.Float64(); err != nil {
			return err
		}
		s.minimum = T(min)
	} else {
		return fmt.Errorf("missing minimum field")
	}

	return nil
}

# Stats

This package implements an _online_ computation of summary statistics meaning that the statistics are computed at each update without requiring the storage of the entire set of samples. This package implements a generic struct that can compute the mean, range (minimum, maximum), standard deviation, and variance of ints, floats, and any type whose underlying type is a float or int (of any size).

NOTE: the structure in this package is not thread-safe and needs to be protected with a mutex if used concurrently across multiple go routines.

## Usage

Create a statistics data structure with the type you'd like to use. Then use the `Update` method to add samples. For example to compute stats for float64s:

```go
myStats := new(stats.Statistics[float64])
myStats.Update(1.31, 3.94, 4.29, 2.30, 7.21)

// Get distribution statistics
myStats.N()
myStats.Mean()
myStats.StdDev()
myStats.Minimum()
myStats.Maximum()
myStats.Range()
```

If you'd like to use a different type, its underlying type must be an int or a float of any size. For example, to compute stats for `time.Distribution`:

```go
times := new(stats.Statistics[time.Duration])
```

## Serialization

You can marshal a statistics object as JSON, which will include the distributional values, e.g.:

```json
{
    "maximum": 4.671499986908815,
    "mean": -0.0004712672972730699,
    "minimum": -4.962035104188279,
    "range": 9.633535091097094,
    "samples": 1000000,
    "squares": 999823.5614339677,
    "stddev": 0.9999121657252907,
    "total": -471.2672972730699,
    "variance": 0.9998243391654413
}
```

However, to unmarshal the object, you only need the `samples`, `total`, `squares`, `minimum`, and `maximum` keys since the other values can be computed with those values.

A statistics object can also be marshaled as a 40 byte array, which is a bit more compact and is what is used to `Value` and `Scan` the statistics into a database object.
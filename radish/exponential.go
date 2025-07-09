package radish

import (
	"math/rand/v2"
	"time"
)

const (
	DefaultInitialInterval     = 500 * time.Millisecond
	DefaultRandomizationFactor = 0.5
	DefaultMultiplier          = 1.5
	DefaultMaxInterval         = 60 * time.Second
	DefaultMaxElapsedTime      = 15 * time.Minute
)

// ExponentialBackOff implements an exponential backoff strategy that increases the
// wait interval after each retry. The next interval is calculated by the formula:
//
// nextInterval = min(currentInterval * rand(1-randomizationFactor, 1+randomizationFactor), maxInterval)
//
// The current interval is multiplied by the multiplier after each retry.
// If maxElapsedTime is set, then the backoff will stop if the total elapsed time has
// exceeded maxElapsedTime.
type ExponentialBackOff struct {
	initialInterval     time.Duration
	randomizationFactor float64
	multiplier          float64
	maxInterval         time.Duration
	maxElapsedTime      time.Duration
	currentInterval     time.Duration
	startTime           time.Time
}

type ExponentialBackOffOption func(*ExponentialBackOff)

func WithInitialInterval(interval time.Duration) ExponentialBackOffOption {
	return func(b *ExponentialBackOff) {
		b.initialInterval = interval
	}
}

func WithRandomizationFactor(factor float64) ExponentialBackOffOption {
	return func(b *ExponentialBackOff) {
		b.randomizationFactor = factor
	}
}

func WithMultiplier(multiplier float64) ExponentialBackOffOption {
	return func(b *ExponentialBackOff) {
		b.multiplier = multiplier
	}
}

func WithMaxInterval(interval time.Duration) ExponentialBackOffOption {
	return func(b *ExponentialBackOff) {
		b.maxInterval = interval
	}
}

func WithMaxElapsedTime(elapsed time.Duration) ExponentialBackOffOption {
	return func(b *ExponentialBackOff) {
		b.maxElapsedTime = elapsed
	}
}

func NewExponentialBackOff(opts ...ExponentialBackOffOption) *ExponentialBackOff {
	backoff := &ExponentialBackOff{
		initialInterval:     DefaultInitialInterval,
		randomizationFactor: DefaultRandomizationFactor,
		multiplier:          DefaultMultiplier,
		maxInterval:         DefaultMaxInterval,
		maxElapsedTime:      DefaultMaxElapsedTime,
	}
	for _, opt := range opts {
		opt(backoff)
	}
	backoff.Reset()
	return backoff
}

func (b *ExponentialBackOff) Next() time.Duration {
	elapsed := b.ElapsedTime()
	next := b.nextInterval()

	// Increase the current interval using the multiplier but ensure it does not exceed
	// the maximum interval.
	b.currentInterval = min(time.Duration(float64(b.currentInterval)*b.multiplier), b.maxInterval)

	// Stop the backoff strategy if the max elapsed time has been exceeded.
	if b.maxElapsedTime > 0 && elapsed+next > b.maxElapsedTime {
		return StopBackOff
	}

	return next
}

func (b *ExponentialBackOff) Reset() {
	b.currentInterval = b.initialInterval
	b.startTime = time.Now()
}

func (b *ExponentialBackOff) ElapsedTime() time.Duration {
	if b.startTime.IsZero() {
		return 0
	}
	return time.Since(b.startTime)
}

// Compute the next backoff interval by randomly selecting a value within the range
// defined by the randomization factor.
func (b *ExponentialBackOff) nextInterval() time.Duration {
	if b.randomizationFactor == 0 {
		return b.currentInterval
	}

	delta := b.randomizationFactor * float64(b.currentInterval)
	min := float64(b.currentInterval) - delta
	max := float64(b.currentInterval) + delta

	return time.Duration(min + (max-min+1)*rand.Float64())
}

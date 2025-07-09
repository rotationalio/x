package radish

import "time"

// StopBackOff is returned by Next() when no more retries should be attempted.
const StopBackOff time.Duration = -1

// BackOff is an interface to support different backoff strategies
type BackOff interface {
	Next() time.Duration
	Reset()
}

// ConstantBackOff implements a simple backoff strategy that always returns the same
// wait interval.
type ConstantBackOff struct {
	interval time.Duration
}

func NewConstantBackOff(interval time.Duration) *ConstantBackOff {
	return &ConstantBackOff{
		interval: interval,
	}
}

func (b *ConstantBackOff) Next() time.Duration {
	return b.interval
}

func (b *ConstantBackOff) Reset() {}

// ZeroBackOff retries immediately without any delay.
type ZeroBackOff struct{}

func NewZeroBackOff() *ZeroBackOff {
	return &ZeroBackOff{}
}

func (b *ZeroBackOff) Next() time.Duration {
	return 0
}

func (b *ZeroBackOff) Reset() {}

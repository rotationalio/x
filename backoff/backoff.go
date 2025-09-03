package backoff

import "time"

// BackOff is a backoff policy for retrying operations.
type BackOff interface {
	// NextBackOff returns the duration to wait before the next retry.
	// backoff.Stop is used to indicate no more retries should be made.
	NextBackOff() time.Duration

	// Reset to initial state.
	Reset()
}

// Stop indicates that no more retries should be made for use in NextBackOff()
const Stop time.Duration = -1

// ZeroBackOff is a fixed backoff policy whose backoff time is always zero, meaning
// that the operation should be retried immediately without waiting, indefinitely.
type ZeroBackOff struct{}

func (b *ZeroBackOff) NextBackOff() time.Duration { return 0 }
func (b *ZeroBackOff) Reset()                     {}

// StopBackOff is a fixed backoff policy that always returns backoff.Stop, meaning that
// the operation should never be retried.
type StopBackOff struct{}

func (b *StopBackOff) NextBackOff() time.Duration { return Stop }
func (b *StopBackOff) Reset()                     {}

// ConstantBackOff is a backoff policy that always returns the same backoff delay.
type ConstantBackOff struct {
	Delay time.Duration
}

func (b *ConstantBackOff) NextBackOff() time.Duration { return b.Delay }
func (b *ConstantBackOff) Reset()                     {}

func NewConstantBackOff(delay time.Duration) *ConstantBackOff {
	return &ConstantBackOff{Delay: delay}
}

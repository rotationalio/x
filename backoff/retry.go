package backoff

import (
	"context"
	"errors"
	"time"
)

// Sets a default limit for the total retry duration
const DefaultMaxElapsedTime = 15 * time.Minute

// Operation is a function that may be retried
type Operation[T any] func() (T, error)

// Notify is a function called on operation error with the error and backoff duration.
type Notify func(error, time.Duration)

//===========================================================================
// Retry Handler
//===========================================================================

// Retry attempts the operation until success, a permanent error, or backoff completion.
// It ensures the operation is executed at least once.
//
// Returns the operation result or error if retries are exhausted or context is canceled.
func Retry[T any](ctx context.Context, operation Operation[T], opts ...RetryOption) (T, error) {
	// Initialize default retry options.
	args := &retryOptions{
		backoff:        NewExponentialBackOff(),
		maxElapsedTime: DefaultMaxElapsedTime,
	}

	// Apply user-provided options
	for _, opt := range opts {
		opt(args)
	}

	// Create the timer for waiting between retries
	var timer *time.Timer
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

	// Start retrying the operation
	started := time.Now()
	args.backoff.Reset()

	for attempts := uint(1); ; attempts++ {
		// Execute the operation.
		res, err := operation()
		if err == nil {
			return res, nil
		}

		// Stop retrying if maximum tries exceeded
		if args.maxTries > 0 && attempts >= args.maxTries {
			return res, err
		}

		// If the error is a no retry error, do not retry
		var stop *NoRetryError
		if errors.As(err, &stop) {
			return res, stop.Unwrap()
		}

		// Check if the context has been canceled
		if cerr := context.Cause(ctx); cerr != nil {
			return res, cerr
		}

		// Calculate the next backoff duration.
		delay := args.backoff.NextBackOff()
		if delay == Stop {
			return res, err
		}

		// Stop retrying if maximum elapsed time is exceeded
		if args.maxElapsedTime > 0 && time.Since(started)+delay > args.maxElapsedTime {
			return res, err
		}

		// Notify on error if a notifier function is provided
		if args.notify != nil {
			args.notify(err, delay)
		}

		// Wait for the next backoff period or context cancelation.
		if timer == nil {
			timer = time.NewTimer(delay)
		} else {
			timer.Reset(delay)
		}

		select {
		case <-timer.C:
		case <-ctx.Done():
			return res, context.Cause(ctx)
		}
	}
}

//===========================================================================
// Retry Options
//===========================================================================

type retryOptions struct {
	backoff        BackOff       // strategy for calculating backoff periods
	notify         Notify        // optional function to notify on each retry error
	maxTries       uint          // maximum number of retry attempts
	maxElapsedTime time.Duration // maximum total time for all retries
}

type RetryOption func(*retryOptions)

// WithBackOff configures a custom backoff strategy.
func WithBackOff(b BackOff) RetryOption {
	return func(opts *retryOptions) {
		opts.backoff = b
	}
}

// WithNotify sets a notification function to handle retry errors.
func WithNotify(n Notify) RetryOption {
	return func(opts *retryOptions) {
		opts.notify = n
	}
}

// WithMaxTries limits the number of all attempts.
func WithMaxTries(n uint) RetryOption {
	return func(opts *retryOptions) {
		opts.maxTries = n
	}
}

// WithMaxElapsedTime limits the total time for all attempts.
func WithMaxElapsedTime(d time.Duration) RetryOption {
	return func(opts *retryOptions) {
		opts.maxElapsedTime = d
	}
}

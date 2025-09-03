package radish

import (
	"context"
	"time"

	"go.rtnl.ai/x/backoff"
)

// Options are used to configure the behavior of the task manager.
type Option func(*TaskManager)

// Specify the number of workers to use in the task manager (default 4).
func WithWorkers(workers int) Option {
	return func(o *TaskManager) {
		o.workers = workers
	}
}

// Specify the size of the task queue in the task manager (default 64).
// If the queue is full, the task manager will block until a worker is available.
func WithQueueSize(size int) Option {
	return func(o *TaskManager) {
		o.queueSize = size
	}
}

// Options configure the task beyond the input context allowing for retries or backoff
// delays in task processing when there are failures or other task-specific handling.
type TaskOption func(*TaskHandler)

// Specify the number of times to retry a task when it returns an error (default 0).
func WithRetries(retries int) TaskOption {
	return func(o *TaskHandler) {
		o.retries = retries
	}
}

// Backoff strategy to use when retrying (default constant backoff).
func WithBackOff(backoff backoff.BackOff) TaskOption {
	return func(o *TaskHandler) {
		o.backoff = backoff
	}
}

// Specify a base context to be used as the parent context when the task is executed
// and on all subsequent retries.
//
// NOTE: it is recommended that this context does not contain a deadline, otherwise the
// deadline may expire before the specified number of retries. Use WithTimeout instead.
func WithContext(ctx context.Context) TaskOption {
	return func(o *TaskHandler) {
		o.ctx = ctx
	}
}

// Specify a timeout to add to the context before passing it into the task function.
func WithTimeout(timeout time.Duration) TaskOption {
	return func(o *TaskHandler) {
		o.timeout = timeout
	}
}

// Log a specific error if all retries failed under the provided context. This error
// will be bundled with the errors that caused the retry failure and reported in a
// single error log message.
func WithError(err error) TaskOption {
	return func(o *TaskHandler) {
		o.err = Errorw(err)
	}
}

// Log a specific error as WithError but using fmt.Errorf semantics to create the err.
func WithErrorf(format string, a ...any) TaskOption {
	return func(o *TaskHandler) {
		o.err = Errorf(format, a...)
	}
}

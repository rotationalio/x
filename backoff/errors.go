package backoff

// NoRetryError signals that the operation should not be retried.
type NoRetryError struct {
	Err error
}

// NoRetry wraps the err in a *NoRetryError to prevent retries.
func NoRetry(err error) error {
	if err == nil {
		return nil
	}
	return &NoRetryError{Err: err}
}

func (e *NoRetryError) Error() string {
	return e.Err.Error()
}

func (e *NoRetryError) Unwrap() error {
	return e.Err
}

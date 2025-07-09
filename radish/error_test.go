package radish_test

import (
	"errors"
	"testing"
	"time"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/radish"
)

func TestTaskErrors(t *testing.T) {
	werr := errors.New("significant badness happened")
	err := radish.Errorw(werr)
	assert.True(t, err.Is(werr), "expected the error to match the wrapped error")

	ferr := radish.Errorf("after %d attempts: %w", 0, werr)
	assert.True(t, ferr.Is(werr), "expected the error to match the wrapped error")

	assert.ErrorIs(t, errors.Unwrap(err), werr, "expected to be able to unwrap an error")
	assert.ErrorIs(t, err, werr, "expected the error to wrap an error")
	assert.EqualError(t, err, "after 0 attempts: significant badness happened")

	// Append some errors
	err.Append(errors.New("could not reach database"))
	err.Append(errors.New("could not reach database"))
	err.Append(errors.New("could not reach database"))
	err.Append(errors.New("failed precondition"))
	err.Append(errors.New("maximum backoff limit reached"))

	err.Since(time.Now().Add(-10 * time.Second))
	assert.EqualError(t, err, "after 5 attempts: significant badness happened")

	err = &radish.Error{}
	testErr := errors.New("backoff limit reached")
	err.Append(testErr)
	assert.False(t, err.Is(errors.New("this error did not happen")))
	assert.True(t, err.Is(testErr), "expected the error to match the appended error")
	assert.Equal(t, "task failed after 1 attempts", err.Error(), "wrong error message")
}

func TestNilTaskError(t *testing.T) {
	err := &radish.Error{}

	assert.Nil(t, errors.Unwrap(err))
	assert.EqualError(t, err, "task failed after 0 attempts")

	// Append some errors
	err.Append(errors.New("could not reach database"))
	err.Append(errors.New("could not reach database"))
	err.Append(errors.New("could not reach database"))
	err.Append(errors.New("failed precondition"))
	err.Append(errors.New("maximum backoff limit reached"))

	err.Since(time.Now().Add(-10 * time.Second))
	assert.EqualError(t, err, "task failed after 5 attempts")
}

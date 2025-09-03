package backoff_test

import (
	"errors"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/backoff"
)

func TestNoRetryError(t *testing.T) {
	want := errors.New("foo")
	other := errors.New("got")

	var err error = backoff.NoRetry(want)
	assert.Equal(t, want, errors.Unwrap(err))
	assert.True(t, errors.Is(err, want))
	assert.False(t, errors.Is(err, other))
}

func TestNoRetryNil(t *testing.T) {
	var err error = backoff.NoRetry(nil)
	assert.Nil(t, err)
	assert.Nil(t, errors.Unwrap(err))
}

package radish

import (
	"testing"
	"time"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/backoff"
)

func TestOptions(t *testing.T) {
	makeOptions := func(opts ...TaskOption) *TaskHandler {
		tm := &TaskManager{}
		return tm.WrapTask(nil, opts...)
	}

	t.Run("Defaults", func(t *testing.T) {
		opts := makeOptions()
		assert.Equal(t, 0, opts.retries)
		assert.Nil(t, opts.backoff)
		assert.Equal(t, time.Duration(0), opts.timeout)
		assert.NotNil(t, opts.err)
		assert.False(t, opts.queuedAt.IsZero())
	})

	t.Run("User", func(t *testing.T) {
		opts := makeOptions(
			WithRetries(42),
			WithBackOff(backoff.NewConstantBackOff(1*time.Second)),
			WithTimeout(1*time.Second),
			WithError(ErrTaskManagerStopped),
		)

		assert.Equal(t, 42, opts.retries)
		assert.IsType(t, backoff.NewConstantBackOff(1*time.Second), opts.backoff)
		assert.Equal(t, 1*time.Second, opts.timeout)
		assert.ErrorIs(t, opts.err, ErrTaskManagerStopped)
		assert.False(t, opts.queuedAt.IsZero())
	})

	t.Run("Practical", func(t *testing.T) {
		opts := makeOptions(
			WithRetries(2),
			WithErrorf("%s wicked this way comes", "something"),
		)

		assert.Equal(t, 2, opts.retries)
		assert.IsType(t, backoff.NewExponentialBackOff(), opts.backoff)
		assert.Equal(t, time.Duration(0), opts.timeout)
		assert.EqualError(t, opts.err, "after 0 attempts: something wicked this way comes")
		assert.False(t, opts.queuedAt.IsZero())
	})
}

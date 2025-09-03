package backoff_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/backoff"
)

func TestRetry(t *testing.T) {
	const successOn = 3

	t.Run("Success", func(t *testing.T) {
		var i = 0
		op := func() (bool, error) {
			i++
			t.Logf("function is called %d times", i)

			if i == successOn {
				t.Log("ok")
				return true, nil
			}

			t.Log("error")
			return false, errors.New("error")
		}

		_, err := backoff.Retry(context.Background(), op)
		assert.Ok(t, err)
		assert.Equal(t, i, successOn)
	})

	t.Run("Data", func(t *testing.T) {
		var i = 0
		op := func() (uint64, error) {
			i++
			t.Logf("function is called %d times", i)

			if i == successOn {
				t.Log("ok")
				return 42, nil
			}

			t.Log("error")
			return 0, errors.New("error")
		}

		res, err := backoff.Retry(context.Background(), op)
		assert.Ok(t, err)
		assert.Equal(t, i, successOn)
		assert.Equal(t, uint64(42), res)
	})

	t.Run("CancelContext", func(t *testing.T) {
		var i = 0
		var expected = errors.New("foo")

		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(context.Canceled)

		op := func() (bool, error) {
			i++
			t.Logf("function is called %d times", i)

			if i == successOn {
				t.Log("cancel")
				cancel(expected)
			}

			t.Log("error")
			return false, errors.New("error")
		}

		_, err := backoff.Retry(ctx, op)
		assert.ErrorIs(t, err, expected)
		assert.Equal(t, i, successOn)
	})

	t.Run("Errors", func(t *testing.T) {
		testCases := []struct {
			name     string
			f        func() (int, error)
			retry    bool
			expected int
		}{
			{
				"nil",
				func() (int, error) {
					return 1, nil
				},
				false,
				1,
			},
			{
				"io.EOF",
				func() (int, error) {
					return 2, io.EOF
				},
				true,
				-1,
			},
			{
				"NoRetry(io.EOF)",
				func() (int, error) {
					return 3, backoff.NoRetry(io.EOF)
				},
				false,
				3,
			},
			{
				"Wrapped: NoRetry(io.EOF)",
				func() (int, error) {
					return 4, fmt.Errorf("Wrapped error: %w", backoff.NoRetry(io.EOF))
				},
				false,
				4,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				attempts := -1
				maxTries := 1

				res, _ := backoff.Retry(
					context.Background(),
					func() (int, error) {
						attempts++
						if attempts >= maxTries {
							return -1, backoff.NoRetry(errors.New("forced"))
						}
						return tc.f()
					})

				if tc.retry {
					assert.NotEqual(t, 0, attempts, "backoff should have retried")
				} else {
					assert.Equal(t, 0, attempts, "backoff should not have retried")
				}

				assert.Equal(t, tc.expected, res, "result mismatch")
			})
		}
	})
}

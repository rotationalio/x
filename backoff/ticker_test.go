package backoff_test

import (
	"errors"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/backoff"
)

func TestTicker(t *testing.T) {
	const successOn = 3
	var i = 0

	// This function is successful on "successOn" calls.
	f := func() error {
		i++
		t.Logf("function called %d times\n", i)

		if i == successOn {
			t.Log("ok")
			return nil
		}

		t.Log("error")
		return errors.New("error")
	}

	b := backoff.NewExponentialBackOff()
	ticker := backoff.NewTicker(b)

	var err error
	for range ticker.C {
		if err = f(); err != nil {
			t.Log(err)
			continue
		}

		break
	}

	assert.Ok(t, err)
	assert.Equal(t, successOn, i)
}

package probez_test

import (
	"context"
	"math/rand/v2"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/probez"
)

func TestServerURL(t *testing.T) {
	srv := probez.NewServer()
	err := srv.Serve(":0")
	assert.Ok(t, err)
	defer srv.Shutdown(context.Background())

	uri, err := url.Parse((srv.URL()))
	assert.Ok(t, err)
	assert.Equal(t, "http", uri.Scheme)
	assert.Equal(t, "127.0.0.1", uri.Hostname())
	assert.NotEqual(t, "", uri.Port())
}

func TestHTTPServer(t *testing.T) {
	srv := probez.NewServer()
	err := srv.Serve(":0")
	assert.Ok(t, err)
	defer srv.Shutdown(context.Background())

	probe, err := probez.NewProbe(srv.URL())
	assert.Ok(t, err)

	t.Run("Healthy", func(t *testing.T) {
		srv.Healthy()

		ok, status, err := probe.Healthy(context.Background())
		assert.Ok(t, err)
		assert.Equal(t, http.StatusOK, status)
		assert.True(t, ok)

		ok, status, err = probe.Live(context.Background())
		assert.Ok(t, err)
		assert.Equal(t, http.StatusOK, status)
		assert.True(t, ok)
	})

	t.Run("Unealthy", func(t *testing.T) {
		srv.Unhealthy()

		ok, status, err := probe.Healthy(context.Background())
		assert.Ok(t, err)
		assert.Equal(t, http.StatusServiceUnavailable, status)
		assert.False(t, ok)

		ok, status, err = probe.Live(context.Background())
		assert.Ok(t, err)
		assert.Equal(t, http.StatusServiceUnavailable, status)
		assert.False(t, ok)
	})

	t.Run("Ready", func(t *testing.T) {
		srv.Ready()

		ok, status, err := probe.Ready(context.Background())
		assert.Ok(t, err)
		assert.Equal(t, http.StatusOK, status)
		assert.True(t, ok)
	})

	t.Run("NotReady", func(t *testing.T) {
		srv.NotReady()

		ok, status, err := probe.Ready(context.Background())
		assert.Ok(t, err)
		assert.Equal(t, http.StatusServiceUnavailable, status)
		assert.False(t, ok)
	})
}

func TestZeroServer(t *testing.T) {
	// A zero valued server should still serve but return 503 by default.
	srv := &probez.Server{}
	err := srv.Serve(":0")
	assert.Ok(t, err)
	defer srv.Shutdown(context.Background())

	probe, err := probez.NewProbe(srv.URL())
	assert.Ok(t, err)

	ok, status, err := probe.Live(context.Background())
	assert.Ok(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, status)
	assert.False(t, ok)

	ok, status, err = probe.Ready(context.Background())
	assert.Ok(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, status)
	assert.False(t, ok)

	ok, status, err = probe.Healthy(context.Background())
	assert.Ok(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, status)
	assert.False(t, ok)
}

func TestServerConcurrency(t *testing.T) {
	// Ensure there are no races setting the state of the probe server.
	srv := probez.NewServer()
	err := srv.Serve(":0")
	assert.Ok(t, err)
	defer srv.Shutdown(context.Background())

	// Make requests to induce reads on the server
	probe, err := probez.NewProbe(srv.URL())
	assert.Ok(t, err)

	// Start multiple threads changing the health and readiness of the server
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 16; i++ {
				randSleep(10 * time.Millisecond)
				srv.Ready()
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 16; i++ {
				randSleep(10 * time.Millisecond)
				srv.NotReady()
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 16; i++ {
				randSleep(10 * time.Millisecond)
				srv.Healthy()
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 16; i++ {
				randSleep(10 * time.Millisecond)
				srv.Unhealthy()
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 16; i++ {
				randSleep(10 * time.Millisecond)
				probe.Healthy(context.Background())
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 16; i++ {
				randSleep(10 * time.Millisecond)
				probe.Live(context.Background())
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 16; i++ {
				randSleep(10 * time.Millisecond)
				probe.Ready(context.Background())
			}
		}()
	}

	wg.Wait()
}

func randSleep(max time.Duration) {
	i := rand.Int64N(int64(max)) + int64(10*time.Microsecond)
	time.Sleep(time.Duration(i))
}

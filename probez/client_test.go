package probez_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/probez"
)

func TestWaitForReady(t *testing.T) {
	urls := make([]string, 0, 3)
	handlers := make([]*probez.Handler, 0, 3)
	for i := 0; i < 3; i++ {
		h := probez.New()
		handlers = append(handlers, h)

		srv := httptest.NewServer(h)
		urls = append(urls, srv.URL)
		t.Cleanup(srv.Close)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Have servers come online at different intervals
	var wg sync.WaitGroup
	for i, h := range handlers {
		wg.Add(1)
		go func(i int, h *probez.Handler) {
			defer wg.Done()
			<-time.After(time.Duration(i+1) * 800 * time.Millisecond)
			h.Ready()
		}(i, h)
	}

	wg.Add(1)
	var err error
	go func() {
		defer wg.Done()
		err = probez.WaitForReady(ctx, urls...)
	}()

	wg.Wait()
	assert.Ok(t, err)
}

func TestClient(t *testing.T) {
	h := probez.New()
	srv := httptest.NewTLSServer(h)
	defer srv.Close()

	probe, err := probez.NewProbe(srv.URL, probez.WithClient(srv.Client()))
	assert.Ok(t, err)

	t.Run("Live", func(t *testing.T) {
		ok, status, err := probe.Live(context.Background())
		assert.Ok(t, err)
		assert.Equal(t, http.StatusOK, status)
		assert.True(t, ok)
	})

	t.Run("Healthy", func(t *testing.T) {
		ok, status, err := probe.Healthy(context.Background())
		assert.Ok(t, err)
		assert.Equal(t, http.StatusOK, status)
		assert.True(t, ok)
	})

	t.Run("Ready", func(t *testing.T) {
		ok, status, err := probe.Ready(context.Background())
		assert.Ok(t, err)
		assert.Equal(t, http.StatusServiceUnavailable, status)
		assert.False(t, ok)
	})
}

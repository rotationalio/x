package probez_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/probez"
)

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

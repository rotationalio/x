package probez_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/probez"
)

func TestHandlerState(t *testing.T) {
	handler := probez.New()
	assert.True(t, handler.IsHealthy(), "expected default state to be healthy")
	assert.False(t, handler.IsReady(), "expected default state to be not ready")

	handler.Unhealthy()
	handler.Ready()
	assert.False(t, handler.IsHealthy(), "expected new state to be unhealthy")
	assert.True(t, handler.IsReady(), "expected new state to be ready")

	handler.Healthy()
	handler.NotReady()
	assert.True(t, handler.IsHealthy(), "expected new state to be healthy")
	assert.False(t, handler.IsReady(), "expected new state to be not ready")
}

func TestHealthz(t *testing.T) {
	handler := probez.New()

	t.Run("MethodNotAllowed", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/healthz", nil)
		w := httptest.NewRecorder()
		handler.Healthz(w, r)

		result := w.Result()
		assert.Equal(t, http.StatusMethodNotAllowed, result.StatusCode)
	})

	t.Run("Unhealthy", func(t *testing.T) {
		handler.Unhealthy()

		r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()
		handler.Healthz(w, r)

		result := w.Result()
		assert.Equal(t, http.StatusServiceUnavailable, result.StatusCode)
	})

	t.Run("Healthy", func(t *testing.T) {
		handler.Healthy()

		r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()
		handler.Healthz(w, r)

		result := w.Result()
		assert.Equal(t, http.StatusOK, result.StatusCode)

		data, err := io.ReadAll(result.Body)
		assert.Ok(t, err)
		assert.True(t, bytes.Equal(data, []byte("ok")))
	})

}

func TestReadyz(t *testing.T) {
	handler := probez.New()

	t.Run("MethodNotAllowed", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/readyz", nil)
		w := httptest.NewRecorder()
		handler.Readyz(w, r)

		result := w.Result()
		assert.Equal(t, http.StatusMethodNotAllowed, result.StatusCode)
	})

	t.Run("NotReady", func(t *testing.T) {
		handler.NotReady()

		r := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		w := httptest.NewRecorder()
		handler.Readyz(w, r)

		result := w.Result()
		assert.Equal(t, http.StatusServiceUnavailable, result.StatusCode)
	})

	t.Run("Ready", func(t *testing.T) {
		handler.Ready()

		r := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		w := httptest.NewRecorder()
		handler.Readyz(w, r)

		result := w.Result()
		assert.Equal(t, http.StatusOK, result.StatusCode)

		data, err := io.ReadAll(result.Body)
		assert.Ok(t, err)
		assert.True(t, bytes.Equal(data, []byte("ok")))
	})

}

func TestHandlerMux(t *testing.T) {
	handler := probez.New()
	mux := http.NewServeMux()
	handler.Mux(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := srv.Client()

	t.Run("Livez", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodGet, srv.URL+"/livez", nil)
		assert.Ok(t, err)

		reply, err := client.Do(r)
		assert.Ok(t, err)
		assert.Equal(t, http.StatusOK, reply.StatusCode)
	})

	t.Run("Healthz", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodGet, srv.URL+"/healthz", nil)
		assert.Ok(t, err)

		reply, err := client.Do(r)
		assert.Ok(t, err)
		assert.Equal(t, http.StatusOK, reply.StatusCode)
	})

	t.Run("Readyz", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodGet, srv.URL+"/readyz", nil)
		assert.Ok(t, err)

		reply, err := client.Do(r)
		assert.Ok(t, err)
		assert.Equal(t, http.StatusServiceUnavailable, reply.StatusCode)
	})
}

func TestHandlerServeHTTP(t *testing.T) {
	handler := probez.New()
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := srv.Client()

	t.Run("Livez", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodGet, srv.URL+"/livez", nil)
		assert.Ok(t, err)

		reply, err := client.Do(r)
		assert.Ok(t, err)
		assert.Equal(t, http.StatusOK, reply.StatusCode)
	})

	t.Run("Healthz", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodGet, srv.URL+"/healthz", nil)
		assert.Ok(t, err)

		reply, err := client.Do(r)
		assert.Ok(t, err)
		assert.Equal(t, http.StatusOK, reply.StatusCode)
	})

	t.Run("Readyz", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodGet, srv.URL+"/readyz", nil)
		assert.Ok(t, err)

		reply, err := client.Do(r)
		assert.Ok(t, err)
		assert.Equal(t, http.StatusServiceUnavailable, reply.StatusCode)
	})

	t.Run("NotFound", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodGet, srv.URL+"/", nil)
		assert.Ok(t, err)

		reply, err := client.Do(r)
		assert.Ok(t, err)
		assert.Equal(t, http.StatusNotFound, reply.StatusCode)
	})
}

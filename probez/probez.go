/*
Package probez provides an http handler that reports the health and readiness states of
a container when responding to Kubernetes probes (e.g. requests to the /livez, /healthz,
and /readyz endpoints). This handler can be implemented as a stand alone server in
containers with long running processes or added to existing services to ensure they can
control how Kubernetes views the service state.
*/
package probez

import (
	"io"
	"net/http"
	"sync/atomic"
)

const (
	Healthz = "/healthz"
	Livez   = "/livez"
	Readyz  = "/readyz"

	ok          = "ok"
	contentType = "Content-Type"
	textPlain   = "text/plain"
)

// The Probe Handler manages the health and readiness states of a container. When it is
// first created, it starts in a healthy, but not ready state. Users can mark the probe
// as healthy, unhealthy, ready, or not ready, changing how it responses to HTTP Get
// requests at the /livez, /healthz, and /readyz endpoints respectively.
//
// This handler implements the http.Handler interface, but in common practice, users
// should add the Healthz, Livez, and Readyz http.HandlerFuncs to their own muxer or
// router. If you're using an http.ServeMux you can use the Handler.Mux function to
// automatically addd the routes.
type Handler struct {
	healthy *atomic.Value
	ready   *atomic.Value
}

var _ http.Handler = &Handler{}

// Create a new Handler that is healthy but not ready.
func New() *Handler {
	srv := &Handler{
		healthy: &atomic.Value{},
		ready:   &atomic.Value{},
	}

	srv.Healthy()
	srv.NotReady()

	return srv
}

// Healthy sets the probe state to healthy so that it responds 200 Ok to liveness probes.
func (h *Handler) Healthy() {
	if h.healthy == nil {
		h.healthy = &atomic.Value{}
	}
	h.healthy.Store(true)
}

// NotHealthy sets the probe state to unhealthy so that it responds 503 Unavailable to liveness probes.
func (h *Handler) Unhealthy() {
	if h.healthy == nil {
		h.healthy = &atomic.Value{}
	}
	h.healthy.Store(false)
}

// IsHealthy returns if the Handler is healthy or not
func (h *Handler) IsHealthy() bool {
	if h.healthy == nil {
		return false
	}
	return h.healthy.Load().(bool)
}

// Ready sets the probe state to ready so that it responses 200 Ok to readiness probes.
func (h *Handler) Ready() {
	if h.ready == nil {
		h.ready = &atomic.Value{}
	}
	h.ready.Store(true)
}

// NotReady sets the probe state to not ready so that it responses 503 Unavailable to readiness probes.
func (h *Handler) NotReady() {
	if h.ready == nil {
		h.ready = &atomic.Value{}
	}
	h.ready.Store(false)
}

// IsHealthy returns if the Handler is ready or not
func (h *Handler) IsReady() bool {
	if h.ready == nil {
		return false
	}
	return h.ready.Load().(bool)
}

// Mux adds the routes specified by the probez handler
func (h *Handler) Mux(mux *http.ServeMux) {
	mux.HandleFunc(Livez, h.Healthz)
	mux.HandleFunc(Healthz, h.Healthz)
	mux.HandleFunc(Readyz, h.Readyz)
}

// Healthz implements the Kubernetes liveness check on /healthz and /livez
func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	if h.healthy == nil || !h.healthy.Load().(bool) {
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}

	w.Header().Set(contentType, textPlain)
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, ok)
}

// Readyz implements the Kubernetes readiness check on /readyz
func (h *Handler) Readyz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	if h.ready == nil || !h.ready.Load().(bool) {
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}

	w.Header().Set(contentType, textPlain)
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, ok)
}

// ServeHTTP implements the http.Handler interface. It is not recommended that this
// method is used directly, but rather a muxer is used or the standalone probe server.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case Livez, Healthz:
		h.Healthz(w, r)
	case Readyz:
		h.Readyz(w, r)
	default:
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}
}

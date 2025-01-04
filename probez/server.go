package probez

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
)

// Server is a quick way to get a probez.Handler service up and running, particularly
// for containers that do not serve HTTP requests but are long running.
type Server struct {
	Handler
	srv  *http.Server
	addr net.Addr
	errc <-chan error
}

// Creates a new probez.Server that is healthy but not ready.
func NewServer() *Server {
	srv := &Server{
		Handler: Handler{
			healthy: &atomic.Value{},
			ready:   &atomic.Value{},
		},
	}

	srv.Healthy()
	srv.NotReady()

	return srv
}

// Serve probe requests on the specified port, handling the /livez and /readyz endpoints
// according to the state of the server which can be set by the user. This method runs
// the server in its own go routine; check Error() to see if any server errors have
// been returned.
func (s *Server) Serve(addr string) (err error) {
	mux := http.NewServeMux()
	s.Mux(mux)

	s.srv = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ErrorLog:          nil,
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      2 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	var sock net.Listener
	if sock, err = net.Listen("tcp", addr); err != nil {
		return err
	}

	errc := make(chan error, 1)
	s.addr = sock.Addr()
	s.errc = errc

	go func(errc chan<- error) {
		if err := s.srv.Serve(sock); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- err
		}
	}(errc)

	return nil
}

// Check if there were any errors while serving requests.
func (s *Server) Error() error {
	select {
	case err := <-s.errc:
		return err
	default:
		return nil
	}
}

// Shutdown the http server and stop responding to probe requests.
func (s *Server) Shutdown(ctx context.Context) (err error) {
	if s.srv == nil {
		return nil
	}

	s.srv.SetKeepAlivesEnabled(false)
	if err = s.srv.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}

// Return the URL the server is responding to probe requests on.
func (s *Server) URL() string {
	u := &url.URL{
		Scheme: "http",
		Host:   s.addr.String(),
	}

	if addr, ok := s.addr.(*net.TCPAddr); ok && addr.IP.IsUnspecified() {
		u.Host = fmt.Sprintf("127.0.0.1:%d", addr.Port)
	}
	return u.String()
}

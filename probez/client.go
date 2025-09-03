package probez

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"go.rtnl.ai/x/api"
	"go.rtnl.ai/x/backoff"
)

const (
	userAgent    = "Rotational Probe/v1"
	accept       = "text/plain"
	acceptLang   = "en-US,en"
	acceptEncode = "gzip, deflate, br"
	timeout      = 5 * time.Second
)

var (
	probeMu sync.Once
	client  *http.Client
)

//===========================================================================
// Wait For Ready
//===========================================================================

// Wait for ready polls the readiness endpoint(s) of the specified service until it
// responds with a 200, retrying using exponential backoff or until the context deadline
// is expired. A default maximum deadline of 15 minutes is used to ensure that this
// method does not block indefinitely.
//
// If you need to wait for multiple endpoints, you can specify them all at once;
// readiness checks are run in parallel but all endpoints have to respond ready before
// this function exits.
//
// If you specify a URL without a path, then the /readyz path is appended. Otherwise,
// if there is a path component in the URL, it remains unmodified.
func WaitForReady(ctx context.Context, urls ...string) (err error) {
	if len(urls) == 0 {
		return errors.New("no endpoints specified to probe")
	}

	// Create requests to make for each probe
	requests := make([]*http.Request, len(urls))
	for i, uri := range urls {
		var base *url.URL
		if base, err = url.Parse(uri); err != nil {
			return fmt.Errorf("could not parse endpoint for %s: %w", uri, err)
		}

		if base.Path == "" {
			base.Path = Readyz
		}

		if requests[i], err = NewRequest(ctx, base.String()); err != nil {
			return fmt.Errorf("could not create request for %s: %w", uri, err)
		}
	}

	// Create a closure to execute probes in parallel and return the status
	checkReady := func() (bool, error) {
		var (
			wg    sync.WaitGroup
			ready atomic.Bool
		)

		// Assume ready is true until a probe failure -- a single failure means not ready.
		ready.Store(true)

		// In parallel execute the probe requests
		for _, req := range requests {
			wg.Add(1)
			go func(req *http.Request) {
				defer wg.Done()
				if _, err := Do(req); err != nil {
					ready.Store(false)
				}
			}(req)
		}

		wg.Wait()
		if ready.Load() {
			return true, nil
		}
		return false, errors.New("one or more readiness probes failed")
	}

	_, err = backoff.Retry(ctx, checkReady)
	return err
}

//===========================================================================
// Probe Objects
//===========================================================================

// Probes test the liveness, readiness, and health endpoints as exposed by the Handler.
// This interface can be used for testing or to implement readiness and liveness checks
// for a service that supports the Kubernetes probe interface.
//
// All probe endpoints return a bool that reports the state (e.g. ready or not ready),
// the status code reported by the server, and any errors in making the request.
type Probe interface {
	Live(ctx context.Context) (bool, int, error)
	Ready(ctx context.Context) (bool, int, error)
	Healthy(ctx context.Context) (bool, int, error)
}

// Create a new probe to test the readiness and liveness state of the specified base
// url (as specified by the endpoint, which should not contain a path). Can optionally
// specify WithClient() to use a custom http.Client to make requests.
func NewProbe(endpoint string, opts ...ProbeOption) (_ Probe, err error) {
	p := &probe{}
	for _, opt := range opts {
		if err = opt(p); err != nil {
			return nil, err
		}
	}

	if p.baseURL, err = url.Parse(endpoint); err != nil {
		return nil, err
	}

	return p, nil
}

type ProbeOption func(*probe) error

func WithClient(client *http.Client) ProbeOption {
	return func(p *probe) error {
		p.client = client
		return nil
	}
}

type probe struct {
	client  *http.Client
	baseURL *url.URL
}

var _ Probe = &probe{}

func (p *probe) Live(ctx context.Context) (bool, int, error) {
	return p.do(ctx, Livez)
}

func (p *probe) Ready(ctx context.Context) (bool, int, error) {
	return p.do(ctx, Readyz)
}

func (p *probe) Healthy(ctx context.Context) (bool, int, error) {
	return p.do(ctx, Healthz)
}

func (p *probe) do(ctx context.Context, path string) (z bool, status int, err error) {
	u := p.baseURL.ResolveReference(&url.URL{Path: path})

	var req *http.Request
	if req, err = NewRequest(ctx, u.String()); err != nil {
		return false, 0, err
	}

	var rep *http.Response
	if p.client != nil {
		if rep, err = p.client.Do(req); err != nil {
			return false, 0, err
		}
	} else {
		initClient()
		if rep, err = client.Do(req); err != nil {
			return false, 0, err
		}
	}

	defer rep.Body.Close()
	z = rep.StatusCode >= 200 && rep.StatusCode < 300
	return z, rep.StatusCode, nil
}

//===========================================================================
// HTTP Client Helpers
//===========================================================================

// Executes an http request using the default probe client. If it returns a non-200
// response from the endpoint, it returns a status error.
func Do(req *http.Request) (rep *http.Response, err error) {
	initClient()
	if rep, err = client.Do(req); err != nil {
		return nil, err
	}

	if rep.StatusCode < 200 || rep.StatusCode >= 300 {
		defer rep.Body.Close()

		var body string
		if data, err := io.ReadAll(rep.Body); err == nil {
			body = http.StatusText(rep.StatusCode)
		} else {
			body = string(data)
		}

		err = &api.StatusError{
			StatusCode: rep.StatusCode,
			Reply: api.Reply{
				Error: body,
			},
		}
	}

	return rep, err
}

func NewRequest(ctx context.Context, url string) (req *http.Request, err error) {
	if req, err = http.NewRequestWithContext(ctx, http.MethodGet, url, nil); err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", accept)
	req.Header.Set("Accept-Language", acceptLang)
	req.Header.Set("Accept-Encoding", acceptEncode)

	return req, nil
}

func initClient() {
	probeMu.Do(func() {
		client = &http.Client{
			Transport:     nil,
			CheckRedirect: nil,
			Timeout:       timeout,
		}
	})
}

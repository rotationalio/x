package probez

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

const (
	userAgent    = "Probe/v1"
	accept       = "text/plain"
	acceptEncode = "gzip, deflate, br"

	userAgentHeader    = "User-Agent"
	acceptHeader       = "Accept"
	acceptEncodeHeader = "Accept-Encoding"
)

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

	if p.client == nil {
		p.client = &http.Client{
			Transport:     nil,
			CheckRedirect: nil,
			Timeout:       5 * time.Second,
		}

		if p.client.Jar, err = cookiejar.New(nil); err != nil {
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
	if req, err = http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil); err != nil {
		return false, 0, err
	}

	req.Header.Add(userAgentHeader, userAgent)
	req.Header.Add(acceptHeader, accept)
	req.Header.Add(acceptEncodeHeader, acceptEncode)

	var rep *http.Response
	if rep, err = p.client.Do(req); err != nil {
		return false, 0, err
	}
	defer rep.Body.Close()

	z = rep.StatusCode >= 200 && rep.StatusCode < 300
	return z, rep.StatusCode, nil
}

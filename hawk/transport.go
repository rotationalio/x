package hawk

import (
	"errors"
	"net/http"
	"time"
	"uuid"
)

const (
	HeaderAuthorization = "Authorization"
	HeaderRequestID     = "X-Request-ID"
	HeaderTimestamp     = "X-Timestamp"
)

var (
	defaultHeaders = []string{HeaderRequestID, HeaderTimestamp}
)

// Injects an Authorization header into outgoing requests by implementing the
// http.RoundTripper interface and can be used as client-side middleware.
type Transport struct {
	Credentials Credentialing
	Next        http.RoundTripper
	Headers     []string
}

func (t *Transport) RoundTrip(req *http.Request) (rep *http.Response, err error) {
	var (
		keyID string
		key   []byte
		auth  string
	)

	keyID = t.Credentials.KeyID(req)
	if key, err = t.Credentials.Key(keyID); err != nil {
		return nil, closeBody(req, err)
	}

	// Clone the request so that headers are only added to the request that is sent;
	// the http.RoundTripper interface requires that the original is not modified.
	// cSpell:ignore reqc
	reqc := req.Clone(req.Context())

	mac := New(keyID, key)
	headers := t.Headers
	if len(headers) == 0 {
		headers = defaultHeaders

		// Add the default headers to the request if they are not already present.
		if reqc.Header.Get(HeaderRequestID) == "" {
			reqc.Header.Set(HeaderRequestID, uuid.New().String())
		}

		if reqc.Header.Get(HeaderTimestamp) == "" {
			reqc.Header.Set(HeaderTimestamp, time.Now().Format(time.RFC3339))
		}
	}

	for _, header := range headers {
		mac.Append(header, reqc.Header.Get(header))
	}

	if auth, err = mac.Authorization(); err != nil {
		return nil, closeBody(req, err)
	}

	reqc.Header.Set(HeaderAuthorization, auth)
	return t.next(reqc)
}

// Closes the body of a request that will not be sent to the next round tripper; the
// http.RoundTripper interface requires that the body is always closed, including on
// errors. An error closing the body is joined to the error that is returned.
func closeBody(req *http.Request, err error) error {
	if req.Body != nil {
		if cerr := req.Body.Close(); cerr != nil {
			return errors.Join(err, cerr)
		}
	}
	return err
}

func (t *Transport) next(req *http.Request) (*http.Response, error) {
	if t.Next == nil {
		return http.DefaultTransport.RoundTrip(req)
	}
	return t.Next.RoundTrip(req)
}

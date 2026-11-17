package hawk_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/hawk"
)

const credAPIKey = hawk.APIKey(credKeyID + ":" + credSecret)

var errUnverified = errors.New("signature does not match the request headers")

// Authenticates the request as a server would: the token is parsed from the
// Authorization header and verified against the headers of the request.
func authenticate(req *http.Request, key []byte) (token *hawk.Token, err error) {
	if token, err = hawk.Parse(req.Header.Get(hawk.HeaderAuthorization)); err != nil {
		return nil, err
	}

	token.Collect(req.Header)

	var ok bool
	if ok, err = token.Verify(key); err != nil {
		return nil, err
	}

	if !ok {
		return nil, errUnverified
	}
	return token, nil
}

// A round tripper that records the requests it receives instead of sending them.
type recorder struct {
	requests []*http.Request
	reply    *http.Response
	err      error
}

func (r *recorder) RoundTrip(req *http.Request) (*http.Response, error) {
	r.requests = append(r.requests, req)
	if r.err != nil {
		return nil, r.err
	}

	if r.reply != nil {
		return r.reply, nil
	}
	return &http.Response{StatusCode: http.StatusNoContent, Body: http.NoBody, Request: req}, nil
}

// Credentials that record how they are used by the transport.
type stubCredentials struct {
	keyID     string
	key       []byte
	err       error
	request   *http.Request
	requested []string
}

func (s *stubCredentials) KeyID(request *http.Request) string {
	s.request = request
	return s.keyID
}

func (s *stubCredentials) Key(keyID string) ([]byte, error) {
	s.requested = append(s.requested, keyID)
	return s.key, s.err
}

// A request body that records how many times it has been closed.
type trackedBody struct {
	io.Reader
	closed int
	err    error
}

func (b *trackedBody) Close() error {
	b.closed++
	return b.err
}

func newRequest(t *testing.T) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "https://example.com/webhook", nil)
	assert.Ok(t, err, "could not create request")
	return req
}

// Sends the request through a transport that signs the specified headers and returns
// the request that the transport passed on to the next round tripper.
func sign(t *testing.T, headers []string, req *http.Request) *http.Request {
	t.Helper()

	next := &recorder{}
	transport := &hawk.Transport{Credentials: credAPIKey, Next: next, Headers: headers}

	rep, err := transport.RoundTrip(req)
	assert.Ok(t, err, "could not round trip request")
	assert.NotNil(t, rep, "expected a response")
	assert.Len(t, next.requests, 1, "expected one request to be sent")
	return next.requests[0]
}

func TestTransport(t *testing.T) {
	t.Run("DefaultHeaders", func(t *testing.T) {
		// The request ID and timestamp are generated and signed if they are missing.
		before := time.Now().Truncate(time.Second)
		sent := sign(t, nil, newRequest(t))

		_, err := uuid.Parse(sent.Header.Get(hawk.HeaderRequestID))
		assert.Ok(t, err, "request ID is not a UUID")

		ts, err := time.Parse(time.RFC3339, sent.Header.Get(hawk.HeaderTimestamp))
		assert.Ok(t, err, "timestamp is not RFC3339")
		assert.False(t, ts.Before(before), "timestamp %s is before the request was sent", ts)
		assert.False(t, ts.After(time.Now()), "timestamp %s is after the request was sent", ts)

		token, err := authenticate(sent, credKey(t))
		assert.Ok(t, err, "could not authenticate request")
		assert.Equal(t, credKeyID, token.KeyID(), "key ID does not match")
		assert.Equal(t, []string{"x-request-id", "x-timestamp"}, token.Headers(), "signed headers do not match")
	})

	t.Run("ExistingHeaders", func(t *testing.T) {
		// Default headers that are already on the request must not be replaced.
		const (
			requestID = "f17c9693-c280-4836-b544-245d832a11e0"
			timestamp = "2025-05-01T12:30:00Z"
		)

		tests := []struct {
			name      string
			requestID string
			timestamp string
		}{
			{"both", requestID, timestamp},
			{"request ID only", requestID, ""},
			{"timestamp only", "", timestamp},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				req := newRequest(t)
				if tc.requestID != "" {
					req.Header.Set(hawk.HeaderRequestID, tc.requestID)
				}

				if tc.timestamp != "" {
					req.Header.Set(hawk.HeaderTimestamp, tc.timestamp)
				}

				sent := sign(t, nil, req)
				assert.NotEqual(t, "", sent.Header.Get(hawk.HeaderRequestID), "expected a request ID")
				assert.NotEqual(t, "", sent.Header.Get(hawk.HeaderTimestamp), "expected a timestamp")

				if tc.requestID != "" {
					assert.Equal(t, tc.requestID, sent.Header.Get(hawk.HeaderRequestID), "request ID was replaced")
				}

				if tc.timestamp != "" {
					assert.Equal(t, tc.timestamp, sent.Header.Get(hawk.HeaderTimestamp), "timestamp was replaced")
				}

				_, err := authenticate(sent, credKey(t))
				assert.Ok(t, err, "could not authenticate request")
			})
		}
	})

	t.Run("CustomHeaders", func(t *testing.T) {
		// Only the configured headers are signed, in order and without duplicates, and
		// the default headers are not added to the request.
		headers := []string{"X-Transfer-ID", "x-transfer-timestamp", "X-TRANSFER-ID", "X-Missing"}

		req := newRequest(t)
		req.Header.Set("X-Transfer-ID", "f17c9693-c280-4836-b544-245d832a11e0")
		req.Header.Set("X-Transfer-Timestamp", "2025-05-01T12:30:00Z")
		req.Header.Set("X-Unsigned", "original")

		sent := sign(t, headers, req)
		assert.Equal(t, "", sent.Header.Get(hawk.HeaderRequestID), "unexpected request ID")
		assert.Equal(t, "", sent.Header.Get(hawk.HeaderTimestamp), "unexpected timestamp")
		assert.Equal(t, "", sent.Header.Get("X-Missing"), "missing header should not be added")

		token, err := authenticate(sent, credKey(t))
		assert.Ok(t, err, "could not authenticate request")
		assert.Equal(t, credKeyID, token.KeyID(), "key ID does not match")
		assert.Equal(t, []string{"x-transfer-id", "x-transfer-timestamp", "x-missing"}, token.Headers(), "signed headers do not match")

		// A header that is not signed can be modified without detection.
		sent.Header.Set("X-Unsigned", "modified")
		_, err = authenticate(sent, credKey(t))
		assert.Ok(t, err, "unsigned header should not be authenticated")

		// A signed header cannot, including one that was missing when it was signed.
		for _, header := range []string{"X-Transfer-ID", "X-Transfer-Timestamp", "X-Missing"} {
			tampered := sent.Clone(sent.Context())
			tampered.Header.Set(header, "modified")

			_, err = authenticate(tampered, credKey(t))
			assert.ErrorIs(t, err, errUnverified, "modified %s header was authenticated", header)
		}
	})

	t.Run("WrongKey", func(t *testing.T) {
		sent := sign(t, nil, newRequest(t))

		key, _ := credKeyOfLength(t, 64)
		_, err := authenticate(sent, key[16:48])
		assert.ErrorIs(t, err, errUnverified, "request was authenticated with the wrong key")
	})

	t.Run("Authorization", func(t *testing.T) {
		// The token replaces any authorization on the request that is sent, but the
		// authorization of the original request is left as it was.
		req := newRequest(t)
		req.Header.Set(hawk.HeaderAuthorization, "Bearer token")

		sent := sign(t, nil, req)
		values := sent.Header.Values(hawk.HeaderAuthorization)
		assert.Len(t, values, 1, "expected one authorization header")
		assert.True(t, strings.HasPrefix(values[0], "HMAC "), "expected HMAC authorization")
		assert.Equal(t, []string{"Bearer token"}, req.Header.Values(hawk.HeaderAuthorization), "original authorization was modified")

		req = newRequest(t)
		sign(t, nil, req)
		assert.Len(t, req.Header.Values(hawk.HeaderAuthorization), 0, "original request was authorized")
	})

	t.Run("OriginalRequest", func(t *testing.T) {
		// A round tripper must not modify the request: the headers that are added by
		// the transport are only added to the request that is sent.
		tests := []struct {
			name     string
			headers  []string
			existing http.Header
		}{
			{"default headers", nil, http.Header{}},
			{"existing request ID", nil, http.Header{"X-Request-Id": {"f17c9693-c280-4836-b544-245d832a11e0"}}},
			{"existing timestamp", nil, http.Header{"X-Timestamp": {"2025-05-01T12:30:00Z"}}},
			{"custom headers", []string{"X-Transfer-ID", "X-Missing"}, http.Header{"X-Transfer-Id": {"f17c9693-c280-4836-b544-245d832a11e0"}}},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				req := newRequest(t)
				req.Header = tc.existing.Clone()

				// Send the same request twice as it would be if it was retried.
				for i := range 2 {
					sent := sign(t, tc.headers, req)
					assert.Equal(t, tc.existing, req.Header, "call %d: headers of the original request were modified", i)

					_, err := authenticate(sent, credKey(t))
					assert.Ok(t, err, "call %d: could not authenticate request", i)
				}
			})
		}

		// A request that is sent again must not reuse the generated request ID.
		req := newRequest(t)
		first := sign(t, nil, req).Header.Get(hawk.HeaderRequestID)
		second := sign(t, nil, req).Header.Get(hawk.HeaderRequestID)
		assert.NotEqual(t, "", first, "expected a request ID")
		assert.NotEqual(t, "", second, "expected a request ID")
		assert.NotEqual(t, first, second, "request ID was reused")
	})

	t.Run("Body", func(t *testing.T) {
		// The body is left for the next round tripper to read and close.
		body := &trackedBody{Reader: strings.NewReader(`{"hello": "world"}`)}
		req, err := http.NewRequest(http.MethodPost, "https://example.com/webhook", body)
		assert.Ok(t, err, "could not create request")

		sent := sign(t, nil, req)
		assert.Equal(t, 0, body.closed, "body was closed before it was sent")

		data, err := io.ReadAll(sent.Body)
		assert.Ok(t, err, "could not read body")
		assert.Equal(t, `{"hello": "world"}`, string(data), "body does not match")
	})

	t.Run("BodyClosedOnError", func(t *testing.T) {
		// A round tripper must always close the body, including on errors; if the
		// request is not sent then there is no next round tripper to close it.
		errStub := errors.New("credentials unavailable")
		errClose := errors.New("could not close body")

		tests := []struct {
			name  string
			creds hawk.Credentialing
			err   error
			close error
		}{
			{"stub error", &stubCredentials{keyID: credKeyID, err: errStub}, errStub, nil},
			{"api key", hawk.APIKey(credKeyID + credSecret), hawk.ErrInvalidSecret, nil},
			{"credential", &hawk.Credential{Secret: credSecret}, hawk.ErrMissingKeyID, nil},
			{"close error", &stubCredentials{keyID: credKeyID, err: errStub}, errStub, errClose},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				body := &trackedBody{Reader: strings.NewReader(`{"hello": "world"}`), err: tc.close}
				req, err := http.NewRequest(http.MethodPost, "https://example.com/webhook", body)
				assert.Ok(t, err, "could not create request")

				next := &recorder{}
				transport := &hawk.Transport{Credentials: tc.creds, Next: next}

				rep, err := transport.RoundTrip(req)
				assert.ErrorIs(t, err, tc.err, "unexpected error")
				assert.Nil(t, rep, "expected no response")
				assert.Len(t, next.requests, 0, "request was sent without being signed")
				assert.Equal(t, 1, body.closed, "expected body to be closed once")

				// An error closing the body must not be discarded.
				if tc.close != nil {
					assert.ErrorIs(t, err, tc.close, "expected error closing the body")
				}
			})
		}
	})

	t.Run("Nonce", func(t *testing.T) {
		// Signing the same headers twice must not produce the same token.
		headers := []string{"X-Transfer-ID"}
		tokens := make(map[string]struct{})

		for range 8 {
			req := newRequest(t)
			req.Header.Set("X-Transfer-ID", "f17c9693-c280-4836-b544-245d832a11e0")

			sent := sign(t, headers, req)
			tokens[sent.Header.Get(hawk.HeaderAuthorization)] = struct{}{}
		}

		assert.Len(t, tokens, 8, "expected a unique token for every request")
	})

	t.Run("Request", func(t *testing.T) {
		// The rest of the request is passed on to the next round tripper as it was.
		type contextKey struct{}
		ctx := context.WithValue(context.Background(), contextKey{}, "value")

		req, err := http.NewRequestWithContext(ctx, http.MethodPut, "https://example.com/webhook?id=42", strings.NewReader(`{"hello": "world"}`))
		assert.Ok(t, err, "could not create request")
		req.Header.Set("Content-Type", "application/json")

		sent := sign(t, nil, req)
		assert.True(t, sent != req, "expected the request to be cloned")
		assert.Equal(t, http.MethodPut, sent.Method, "method does not match")
		assert.Equal(t, "https://example.com/webhook?id=42", sent.URL.String(), "url does not match")
		assert.Equal(t, "application/json", sent.Header.Get("Content-Type"), "content type does not match")
		assert.Equal(t, "value", sent.Context().Value(contextKey{}), "context does not match")

		body, err := io.ReadAll(sent.Body)
		assert.Ok(t, err, "could not read body")
		assert.Equal(t, `{"hello": "world"}`, string(body), "body does not match")
	})

	t.Run("Response", func(t *testing.T) {
		// The response of the next round tripper is returned without modification.
		reply := &http.Response{StatusCode: http.StatusTeapot, Body: http.NoBody}
		next := &recorder{reply: reply}
		transport := &hawk.Transport{Credentials: credAPIKey, Next: next}

		rep, err := transport.RoundTrip(newRequest(t))
		assert.Ok(t, err, "could not round trip request")
		assert.True(t, rep == reply, "expected the response of the next round tripper")
	})

	t.Run("ResponseError", func(t *testing.T) {
		// The error of the next round tripper is returned without a response.
		errNext := errors.New("connection refused")
		next := &recorder{err: errNext}
		transport := &hawk.Transport{Credentials: credAPIKey, Next: next}

		rep, err := transport.RoundTrip(newRequest(t))
		assert.ErrorIs(t, err, errNext, "unexpected error")
		assert.Nil(t, rep, "expected no response")
		assert.Len(t, next.requests, 1, "expected one request to be sent")
	})

	t.Run("Credentials", func(t *testing.T) {
		// The key ID is requested for the request, the key is requested for that key
		// ID, and both are used to sign the request.
		key, _ := credKeyOfLength(t, 64)
		creds := &stubCredentials{keyID: "stub-key-id", key: key}
		next := &recorder{}
		transport := &hawk.Transport{Credentials: creds, Next: next}

		req := newRequest(t)
		_, err := transport.RoundTrip(req)
		assert.Ok(t, err, "could not round trip request")
		assert.True(t, creds.request == req, "expected key ID to be requested for the request")
		assert.Equal(t, []string{"stub-key-id"}, creds.requested, "expected key to be requested for the key ID")
		assert.Len(t, next.requests, 1, "expected one request to be sent")

		token, err := authenticate(next.requests[0], key)
		assert.Ok(t, err, "could not authenticate request")
		assert.Equal(t, "stub-key-id", token.KeyID(), "key ID does not match")
	})

	t.Run("CredentialErrors", func(t *testing.T) {
		// A request that cannot be signed must not be sent.
		errStub := errors.New("credentials unavailable")
		_, shortSecret := credKeyOfLength(t, 31)

		tests := []struct {
			name  string
			creds hawk.Credentialing
			err   error
		}{
			{"stub error", &stubCredentials{keyID: credKeyID, err: errStub}, errStub},
			{"empty api key", hawk.APIKey(""), hawk.ErrInvalidSecret},
			{"api key without colon", hawk.APIKey(credKeyID + credSecret), hawk.ErrInvalidSecret},
			{"api key without key ID", hawk.APIKey(":" + credSecret), hawk.ErrMissingKeyID},
			{"api key with short secret", hawk.APIKey(credKeyID + ":" + shortSecret), hawk.ErrInvalidSecret},
			{"zero credential", &hawk.Credential{}, hawk.ErrMissingKeyID},
			{"credential without client ID", &hawk.Credential{Secret: credSecret}, hawk.ErrMissingKeyID},
			{"credential without secret", &hawk.Credential{ClientID: credKeyID}, hawk.ErrInvalidSecret},
			{"credential with short secret", &hawk.Credential{ClientID: credKeyID, Secret: shortSecret}, hawk.ErrInvalidSecret},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				next := &recorder{}
				transport := &hawk.Transport{Credentials: tc.creds, Next: next}

				// The error must be returned every time, not just the first time.
				for i := range 2 {
					rep, err := transport.RoundTrip(newRequest(t))
					assert.ErrorIs(t, err, tc.err, "call %d: unexpected error", i)
					assert.Nil(t, rep, "call %d: expected no response", i)
					assert.Len(t, next.requests, 0, "call %d: request was sent without being signed", i)
				}
			})
		}
	})

	t.Run("Concurrent", func(t *testing.T) {
		// A round tripper must be safe for concurrent use by multiple goroutines.
		const workers = 16

		tests := map[string]hawk.Credentialing{
			"APIKey":     credAPIKey,
			"Credential": &hawk.Credential{ClientID: credKeyID, Secret: credSecret},
		}

		for name, creds := range tests {
			t.Run(name, func(t *testing.T) {
				var mu sync.Mutex
				tokens := make(map[string]struct{})

				key := credKey(t)
				next := roundTripFunc(func(req *http.Request) (*http.Response, error) {
					if _, err := authenticate(req, key); err != nil {
						return nil, err
					}

					mu.Lock()
					tokens[req.Header.Get(hawk.HeaderAuthorization)] = struct{}{}
					mu.Unlock()
					return &http.Response{StatusCode: http.StatusNoContent, Body: http.NoBody, Request: req}, nil
				})

				transport := &hawk.Transport{Credentials: creds, Next: next}
				requests := make([]*http.Request, workers)
				for i := range requests {
					requests[i] = newRequest(t)
				}

				var wg sync.WaitGroup
				errs := make([]error, workers)
				for i := range workers {
					wg.Go(func() {
						_, errs[i] = transport.RoundTrip(requests[i])
					})
				}
				wg.Wait()

				for i, err := range errs {
					assert.Ok(t, err, "worker %d: could not round trip request", i)
				}
				assert.Len(t, tokens, workers, "expected a unique token for every request")
			})
		}
	})
}

func TestTransportClient(t *testing.T) {
	// Without a next round tripper the request is sent with the default transport, so
	// an http client with the transport must be authenticated by a real server.
	key := credKey(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := authenticate(r, key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		w.Header().Set("X-Key-ID", token.KeyID())
		w.Header().Set("X-Signed-Headers", strings.Join(token.Headers(), ";"))
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	otherKey, otherSecret := credKeyOfLength(t, 64)
	assert.NotEqual(t, key, otherKey[16:48], "expected a different key")

	tests := []struct {
		name      string
		transport http.RoundTripper
		status    int
		signed    string
	}{
		{
			"no transport", nil,
			http.StatusUnauthorized, "",
		},
		{
			"api key", &hawk.Transport{Credentials: credAPIKey},
			http.StatusNoContent, "x-request-id;x-timestamp",
		},
		{
			"credential", &hawk.Transport{Credentials: &hawk.Credential{ClientID: credKeyID, Secret: credSecret}},
			http.StatusNoContent, "x-request-id;x-timestamp",
		},
		{
			"custom headers", &hawk.Transport{Credentials: credAPIKey, Headers: []string{"Content-Type", "X-Transfer-ID"}},
			http.StatusNoContent, "content-type;x-transfer-id",
		},
		{
			"wrong secret", &hawk.Transport{Credentials: hawk.APIKey(credKeyID + ":" + otherSecret)},
			http.StatusUnauthorized, "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &http.Client{Transport: tc.transport}

			req, err := http.NewRequest(http.MethodPost, srv.URL+"/webhook", strings.NewReader(`{"hello": "world"}`))
			assert.Ok(t, err, "could not create request")
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Transfer-ID", "f17c9693-c280-4836-b544-245d832a11e0")

			rep, err := client.Do(req)
			assert.Ok(t, err, "could not execute request")
			t.Cleanup(func() { rep.Body.Close() })

			assert.Equal(t, tc.status, rep.StatusCode, "unexpected status code")
			if tc.status == http.StatusNoContent {
				assert.Equal(t, credKeyID, rep.Header.Get("X-Key-ID"), "key ID does not match")
				assert.Equal(t, tc.signed, rep.Header.Get("X-Signed-Headers"), "signed headers do not match")
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

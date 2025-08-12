package httpcc_test

import (
	"net/http"
	"testing"
	"time"

	"go.rtnl.ai/x/assert"
	. "go.rtnl.ai/x/httpcc"
)

func TestRequest(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
		assert.Ok(t, err, "failed to create request")
		directive, err := Request(*req)
		assert.Ok(t, err, "failed to parse request cache control")

		maxAge, ok := directive.MaxAge()
		assert.False(t, ok)
		assert.Equal(t, uint64(0), maxAge)

		assert.False(t, directive.NoCache())
		assert.False(t, directive.NoStore())
		assert.False(t, directive.NoTransform())

		staleIfError, ok := directive.StaleIfError()
		assert.False(t, ok)
		assert.Equal(t, uint64(0), staleIfError)

		extension := directive.Extensions()
		assert.Len(t, extension, 0)

		test := directive.Extension("nonexistent")
		assert.Equal(t, "", test)

		ifNoneMatch, ok := directive.IfNoneMatch()
		assert.False(t, ok)
		assert.Equal(t, "", ifNoneMatch)

		ifUnmodifiedSince, ok := directive.IfUnmodifiedSince()
		assert.False(t, ok)
		assert.True(t, ifUnmodifiedSince.IsZero())

		ifModifiedSince, ok := directive.IfModifiedSince()
		assert.False(t, ok)
		assert.True(t, ifModifiedSince.IsZero())

		maxStale, ok := directive.MaxStale()
		assert.False(t, ok)
		assert.Equal(t, uint64(0), maxStale)

		minFresh, ok := directive.MinFresh()
		assert.False(t, ok)
		assert.Equal(t, uint64(0), minFresh)

		assert.False(t, directive.OnlyIfCached())
	})

	t.Run("NoCacheControl", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
		assert.Ok(t, err, "failed to create request")

		ts := time.Date(2025, 4, 7, 13, 21, 43, 0, time.UTC)

		req.Header.Set(IfNoneMatch, "myetag")
		req.Header.Set(IfUnmodifiedSince, ts.Format(http.TimeFormat))
		req.Header.Set(IfModifiedSince, ts.Format(http.TimeFormat))

		directive, err := Request(*req)
		assert.Ok(t, err, "failed to parse request cache control")

		ifNoneMatch, ok := directive.IfNoneMatch()
		assert.True(t, ok)
		assert.Equal(t, "myetag", ifNoneMatch)

		ifUnmodifiedSince, ok := directive.IfUnmodifiedSince()
		assert.True(t, ok)
		assert.Equal(t, ts, ifUnmodifiedSince)

		ifModifiedSince, ok := directive.IfModifiedSince()
		assert.True(t, ok)
		assert.Equal(t, ts, ifModifiedSince)

		maxAge, ok := directive.MaxAge()
		assert.False(t, ok)
		assert.Equal(t, uint64(0), maxAge)
	})

	t.Run("CacheControl", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
		assert.Ok(t, err, "failed to create request")

		ts := time.Date(2025, 4, 7, 13, 21, 43, 0, time.UTC)

		req.Header.Set(CacheControl, "max-age=604800, no-transform, stale-if-error=86400")
		req.Header.Set(IfNoneMatch, "myetag")
		req.Header.Set(IfUnmodifiedSince, ts.Format(http.TimeFormat))
		req.Header.Set(IfModifiedSince, ts.Format(http.TimeFormat))

		directive, err := Request(*req)
		assert.Ok(t, err, "failed to parse request cache control")

		maxAge, ok := directive.MaxAge()
		assert.True(t, ok)
		assert.Equal(t, uint64(604800), maxAge)

		assert.False(t, directive.NoCache())
		assert.False(t, directive.NoStore())
		assert.True(t, directive.NoTransform())

		staleIfError, ok := directive.StaleIfError()
		assert.True(t, ok)
		assert.Equal(t, uint64(86400), staleIfError)

		extension := directive.Extensions()
		assert.Len(t, extension, 0)

		ifNoneMatch, ok := directive.IfNoneMatch()
		assert.True(t, ok)
		assert.Equal(t, "myetag", ifNoneMatch)

		ifUnmodifiedSince, ok := directive.IfUnmodifiedSince()
		assert.True(t, ok)
		assert.Equal(t, ts, ifUnmodifiedSince)

		ifModifiedSince, ok := directive.IfModifiedSince()
		assert.True(t, ok)
		assert.Equal(t, ts, ifModifiedSince)

		// Quick pass-through test
		dir2, err := Request(directive)
		assert.Ok(t, err, "failed to parse request cache control from directive")
		assert.Equal(t, directive, dir2, "directive did not match original")
	})

	t.Run("BadType", func(t *testing.T) {
		_, err := Request(1233)
		assert.EqualError(t, err, "unsupported type int for request cache control parsing")
	})

	t.Run("IfNoneMatch", func(t *testing.T) {
		t.Run("Quoted", func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
			assert.Ok(t, err, "failed to create request")
			req.Header.Set(IfNoneMatch, `"myetag"`)

			directive, err := Request(*req)
			assert.Ok(t, err, "failed to parse request cache control")

			ifNoneMatch, ok := directive.IfNoneMatch()
			assert.True(t, ok)
			assert.Equal(t, "myetag", ifNoneMatch)
		})

		t.Run("Unquoted", func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
			assert.Ok(t, err, "failed to create request")
			req.Header.Set(IfNoneMatch, "myetag")

			directive, err := Request(*req)
			assert.Ok(t, err, "failed to parse request cache control")

			ifNoneMatch, ok := directive.IfNoneMatch()
			assert.True(t, ok)
			assert.Equal(t, "myetag", ifNoneMatch)
		})
	})
}

func TestResponse(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		rep := &http.Response{}
		directive, err := Response(*rep)
		assert.Ok(t, err, "failed to parse response cache control")

		maxAge, ok := directive.MaxAge()
		assert.False(t, ok)
		assert.Equal(t, uint64(0), maxAge)

		assert.False(t, directive.NoCache())
		assert.False(t, directive.NoStore())
		assert.False(t, directive.NoTransform())

		staleIfError, ok := directive.StaleIfError()
		assert.False(t, ok)
		assert.Equal(t, uint64(0), staleIfError)

		extension := directive.Extensions()
		assert.Len(t, extension, 0)

		test := directive.Extension("nonexistent")
		assert.Equal(t, "", test)

		expires, ok := directive.Expires()
		assert.False(t, ok)
		assert.True(t, expires.IsZero())

		lastModified, ok := directive.LastModified()
		assert.False(t, ok)
		assert.True(t, lastModified.IsZero())

		etag, ok := directive.ETag()
		assert.False(t, ok)
		assert.Equal(t, "", etag)

		sMaxAge, ok := directive.SMaxAge()
		assert.False(t, ok)
		assert.Equal(t, uint64(0), sMaxAge)

		assert.False(t, directive.MustRevalidate())
		assert.False(t, directive.ProxyRevalidate())
		assert.False(t, directive.MustUnderstand())
		assert.False(t, directive.Private())
		assert.False(t, directive.Public())
		assert.False(t, directive.Immutable())

		staleWhileRevalidate, ok := directive.StaleWhileRevalidate()
		assert.False(t, ok)
		assert.Equal(t, uint64(0), staleWhileRevalidate)
	})

	t.Run("BadType", func(t *testing.T) {
		_, err := Response(1233)
		assert.EqualError(t, err, "unsupported type int for response cache control parsing")
	})

	t.Run("NoCacheControl", func(t *testing.T) {
		rep := &http.Response{
			Header: make(http.Header),
		}

		ts := time.Date(2025, 4, 7, 13, 21, 43, 0, time.UTC)
		rep.Header.Set(Expires, ts.Format(http.TimeFormat))
		rep.Header.Set(LastModified, ts.Format(http.TimeFormat))
		rep.Header.Set(ETag, "myetag")

		directive, err := Response(*rep)
		assert.Ok(t, err, "failed to parse response cache control")

		expires, ok := directive.Expires()
		assert.True(t, ok)
		assert.Equal(t, ts, expires)

		lastModified, ok := directive.LastModified()
		assert.True(t, ok)
		assert.Equal(t, ts, lastModified)

		etag, ok := directive.ETag()
		assert.True(t, ok)
		assert.Equal(t, "myetag", etag)

		maxAge, ok := directive.MaxAge()
		assert.False(t, ok)
		assert.Equal(t, uint64(0), maxAge)
	})

	t.Run("CacheControl", func(t *testing.T) {
		rep := &http.Response{
			Header: make(http.Header),
		}

		ts := time.Date(2025, 4, 7, 13, 21, 43, 0, time.UTC)

		rep.Header.Set(CacheControl, "max-age=604800, must-revalidate")
		rep.Header.Set(Expires, ts.Format(http.TimeFormat))
		rep.Header.Set(LastModified, ts.Format(http.TimeFormat))
		rep.Header.Set(ETag, "myetag")

		directive, err := Response(*rep)
		assert.Ok(t, err, "failed to parse response cache control")

		maxAge, ok := directive.MaxAge()
		assert.True(t, ok)
		assert.Equal(t, uint64(604800), maxAge)

		assert.True(t, directive.MustRevalidate())

		assert.False(t, directive.NoCache())
		assert.False(t, directive.NoStore())
		assert.False(t, directive.NoTransform())

		staleIfError, ok := directive.StaleIfError()
		assert.False(t, ok)
		assert.Equal(t, uint64(0), staleIfError)

		extension := directive.Extensions()
		assert.Len(t, extension, 0)

		expires, ok := directive.Expires()
		assert.True(t, ok)
		assert.Equal(t, ts, expires)

		lastModified, ok := directive.LastModified()
		assert.True(t, ok)
		assert.Equal(t, ts, lastModified)

		etag, ok := directive.ETag()
		assert.True(t, ok)
		assert.Equal(t, "myetag", etag)

		// Quick pass-through test
		dir2, err := Response(directive)
		assert.Ok(t, err, "failed to parse response cache control from directive")
		assert.Equal(t, directive, dir2)
	})

	t.Run("Etag", func(t *testing.T) {
		t.Run("Quoted", func(t *testing.T) {
			rep := &http.Response{Header: make(http.Header)}
			rep.Header.Set(ETag, `"myetag"`)

			directive, err := Response(*rep)
			assert.Ok(t, err, "failed to parse response cache control")

			etag, ok := directive.ETag()
			assert.True(t, ok)
			assert.Equal(t, "myetag", etag)

			assert.False(t, directive.WeakETag())
		})

		t.Run("Unquoted", func(t *testing.T) {
			rep := &http.Response{Header: make(http.Header)}
			rep.Header.Set(ETag, "myetag")

			directive, err := Response(*rep)
			assert.Ok(t, err, "failed to parse response cache control")

			etag, ok := directive.ETag()
			assert.True(t, ok)
			assert.Equal(t, "myetag", etag)

			assert.False(t, directive.WeakETag())
		})

		t.Run("Weak", func(t *testing.T) {
			rep := &http.Response{Header: make(http.Header)}
			rep.Header.Set(ETag, `W/"myetag"`)

			directive, err := Response(*rep)
			assert.Ok(t, err, "failed to parse response cache control")

			etag, ok := directive.ETag()
			assert.True(t, ok)
			assert.Equal(t, "myetag", etag)

			assert.True(t, directive.WeakETag())
		})
	})
}

func TestParseResponseDirectives(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		var responseDirectives = []struct {
			cc       string
			expected map[string]any
		}{
			{"max-age=604800", map[string]any{"max-age": uint64(604800)}},
			{"max-age=", map[string]any{"max-age": uint64(0)}},
			{"s-maxage=604800", map[string]any{"s-maxage": uint64(604800)}},
			{"no-cache", map[string]any{"no-cache": true}},
			{"max-age=604800, must-revalidate", map[string]any{"max-age": uint64(604800), "must-revalidate": true}},
			{"max-age=604800, proxy-revalidate", map[string]any{"max-age": uint64(604800), "proxy-revalidate": true}},
			{"no-store", map[string]any{"no-store": true}},
			{"private", map[string]any{"private": true}},
			{"private=", map[string]any{"private": true}},
			{"public", map[string]any{"public": true}},
			{"public, max-age=604800", map[string]any{"public": true, "max-age": uint64(604800)}},
			{"must-understand, no-store", map[string]any{"must-understand": true, "no-store": true}},
			{"no-transform,", map[string]any{"no-transform": true}},
			{"public, max-age=604800, immutable", map[string]any{"public": true, "max-age": uint64(604800), "immutable": true}},
			{"max-age=604800, stale-while-revalidate=86400", map[string]any{"max-age": uint64(604800), "stale-while-revalidate": uint64(86400)}},
			{"max-age=604800, stale-if-error=86400", map[string]any{"max-age": uint64(604800), "stale-if-error": uint64(86400)}},
			{"max-age=31536000, immutable", map[string]any{"max-age": uint64(31536000), "immutable": true}},
			{"max-age=0, must-revalidate", map[string]any{"max-age": uint64(0), "must-revalidate": true}},
			{"hello=\"world\"", map[string]any{"hello": "world"}},
			{"ext=\"नमस्ते\"", map[string]any{"ext": "नमस्ते"}},
			{"max-age=604800, ext=\"नमस्ते\"", map[string]any{"max-age": uint64(604800), "ext": "नमस्ते"}},
			{"foo,", map[string]any{"foo": "foo"}},
			{"FOO,", map[string]any{"foo": "FOO"}},
		}

		for _, cc := range responseDirectives {
			directives, err := Response(cc.cc)
			assert.Ok(t, err, "failed to parse response directives for %q", cc.cc)
			assertDirectives(t, directives, cc.expected)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		var testCases = []struct {
			cc   string
			errs string
		}{
			{"max-age=-1", "could not parse max-age: strconv.ParseUint: parsing \"-1\": invalid syntax"},
			{"s-maxage=foo", "could not parse s-maxage: strconv.ParseUint: parsing \"foo\": invalid syntax"},
			{"stale-while-revalidate=foo", "could not parse stale-while-revalidate: strconv.ParseUint: parsing \"foo\": invalid syntax"},
			{"stale-if-error=foo", "could not parse stale-if-error: strconv.ParseUint: parsing \"foo\": invalid syntax"},
			{"public=8600", "failed to parse tokens: invalid value for public (no argument expected)"},
			{"max-age=\"86000\"", "failed to parse tokens: invalid value for max-age (quoted string not allowed)"},
			{"foo=\"notenclosed", "failed to parse tokens: invalid value for foo (unquoting failed: invalid syntax)"},
		}

		for _, tc := range testCases {
			_, err := Response(tc.cc)
			assert.EqualError(t, err, tc.errs, "error message did not contain expected text for %q", tc.cc)
		}
	})
}

func TestParseRequestDirectives(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		var requestDirectives = []struct {
			cc       string
			expected map[string]any
		}{
			{"no-cache", map[string]any{"no-cache": true}},
			{"no-store", map[string]any{"no-store": true}},
			{" max-age=10800 ", map[string]any{"max-age": uint64(10800)}},
			{"max-stale=3600", map[string]any{"max-stale": uint64(3600)}},
			{"min-fresh=600", map[string]any{"min-fresh": uint64(600)}},
			{"no-transform", map[string]any{"no-transform": true}},
			{"only-if-cached", map[string]any{"only-if-cached": true}},
			{"max-age=604800, stale-if-error=86400", map[string]any{"max-age": uint64(604800), "stale-if-error": uint64(86400)}},
			{"hello=\"world\"", map[string]any{"hello": "world"}},
			{"ext=\"नमस्ते\"", map[string]any{"ext": "नमस्ते"}},
			{"max-age=604800, ext=\"नमस्ते\"", map[string]any{"max-age": uint64(604800), "ext": "नमस्ते"}},
			{"foo", map[string]any{"foo": "foo"}},
			{"FOO,", map[string]any{"foo": "FOO"}},
		}

		for _, cc := range requestDirectives {
			directives, err := Request(cc.cc)
			assert.Ok(t, err, "failed to parse request directives for %q", cc.cc)
			assertDirectives(t, directives, cc.expected)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		var testCases = []struct {
			cc   string
			errs string
		}{
			{"max-age=-1", "could not parse max-age: strconv.ParseUint: parsing \"-1\": invalid syntax"},
			{"max-age=\"86000\"", "failed to parse tokens: invalid value for max-age (quoted string not allowed)"},
			{"max-stale=foo", "could not parse max-stale: strconv.ParseUint: parsing \"foo\": invalid syntax"},
			{"min-fresh=foo", "could not parse min-fresh: strconv.ParseUint: parsing \"foo\": invalid syntax"},
			{"only-if-cached=8600", "failed to parse tokens: invalid value for only-if-cached (no argument expected)"},
			{"no-cache=8600", "failed to parse tokens: invalid value for no-cache (no argument expected)"},
			{"stale-if-error=foo", "could not parse stale-if-error: strconv.ParseUint: parsing \"foo\": invalid syntax"},
			{"foo=\"notenclosed", "failed to parse tokens: invalid value for foo (unquoting failed: invalid syntax)"},
		}

		for _, tc := range testCases {
			_, err := Request(tc.cc)
			assert.EqualError(t, err, tc.errs, "error message did not contain expected text for %q", tc.cc)
		}
	})
}

func TestParseEmpty(t *testing.T) {
	t.Run("ResponseDirectives", func(t *testing.T) {
		tokens, err := ParseResponseDirectives("")
		assert.Ok(t, err)
		assert.Len(t, tokens, 0, "Expected no tokens for empty string")
	})

	t.Run("RequestDirectives", func(t *testing.T) {
		tokens, err := ParseRequestDirectives("")
		assert.Ok(t, err)
		assert.Len(t, tokens, 0, "Expected no tokens for empty string")
	})
}

func TestSkipSpaces(t *testing.T) {
	cc := "  \u0085   \u00a0 max-age=604800\u1680 \u2028, \u2000ext=\"नमस्ते\", \u200A \t no-cache \u2003, \v\n \u202F  no-store \f\r \u2029   \u205f \u3000 "

	t.Run("RequestDirective", func(t *testing.T) {
		tokens, err := ParseRequestDirectives(cc)
		assert.Ok(t, err)
		assert.Equal(t, 4, len(tokens), "Expected 4 tokens for %q", cc)
	})

	t.Run("ResponseDirective", func(t *testing.T) {
		tokens, err := ParseResponseDirectives(cc)
		assert.Ok(t, err)
		assert.Equal(t, 4, len(tokens), "Expected 4 tokens for %q", cc)
	})
}

func assertDirectives(t *testing.T, directives any, expected map[string]any) {
	t.Helper()
	for name, expectedValue := range expected {
		assertDirective(t, directives, name, expectedValue)
	}
}

func assertDirective(t *testing.T, directive any, name string, expected any) {
	t.Helper()

	switch d := directive.(type) {
	case *RequestDirective:
		switch name {
		case MaxAge:
			maxAge, ok := d.MaxAge()
			assert.True(t, ok, "expected max-age to be set")
			assert.Equal(t, expected, maxAge, "max-age did not match expected value")
		case NoCache:
			assert.Equal(t, expected, d.NoCache(), "no-cache did not match expected value")
		case NoStore:
			assert.Equal(t, expected, d.NoStore(), "no-store did not match expected value")
		case NoTransform:
			assert.Equal(t, expected, d.NoTransform(), "no-transform did not match expected value")
		case StaleIfError:
			staleIfError, ok := d.StaleIfError()
			assert.True(t, ok, "expected stale-if-error to be set")
			assert.Equal(t, expected, staleIfError, "stale-if-error did not match expected value")
		case IfNoneMatch:
			ifNoneMatch, ok := d.IfNoneMatch()
			assert.True(t, ok, "expected if-none-match to be set")
			assert.Equal(t, expected, ifNoneMatch, "if-none-match did not match expected value")
		case IfUnmodifiedSince:
			ifUnmodifiedSince, ok := d.IfUnmodifiedSince()
			assert.True(t, ok, "expected if-unmodified-since to be set")
			assert.Equal(t, expected, ifUnmodifiedSince, "if-unmodified-since did not match expected value")
		case IfModifiedSince:
			ifModifiedSince, ok := d.IfModifiedSince()
			assert.True(t, ok, "expected if-modified-since to be set")
			assert.Equal(t, expected, ifModifiedSince, "if-modified-since did not match expected value")
		case MaxStale:
			maxStale, ok := d.MaxStale()
			assert.True(t, ok, "expected max-stale to be set")
			assert.Equal(t, expected, maxStale, "max-stale did not match expected value")
		case MinFresh:
			minFresh, ok := d.MinFresh()
			assert.True(t, ok, "expected min-fresh to be set")
			assert.Equal(t, expected, minFresh, "min-fresh did not match expected value")
		case OnlyIfCached:
			assert.Equal(t, expected, d.OnlyIfCached(), "only-if-cached did not match expected value")
		default:
			value := d.Extension(name)
			assert.Equal(t, expected, value, "extension %q did not match expected value", name)
		}
	case *ResponseDirective:
		switch name {
		case MaxAge:
			maxAge, ok := d.MaxAge()
			assert.True(t, ok, "expected max-age to be set")
			assert.Equal(t, expected, maxAge, "max-age did not match expected value")
		case NoCache:
			assert.Equal(t, expected, d.NoCache(), "no-cache did not match expected value")
		case NoStore:
			assert.Equal(t, expected, d.NoStore(), "no-store did not match expected value")
		case NoTransform:
			assert.Equal(t, expected, d.NoTransform(), "no-transform did not match expected value")
		case StaleIfError:
			staleIfError, ok := d.StaleIfError()
			assert.True(t, ok, "expected stale-if-error to be set")
			assert.Equal(t, expected, staleIfError, "stale-if-error did not match expected value")
		case Expires:
			expires, ok := d.Expires()
			assert.True(t, ok, "expected expires to be set")
			assert.Equal(t, expected, expires, "expires did not match expected value")
		case LastModified:
			lastModified, ok := d.LastModified()
			assert.True(t, ok, "expected last-modified to be set")
			assert.Equal(t, expected, lastModified, "last-modified did not match expected value")
		case ETag:
			etag, ok := d.ETag()
			assert.True(t, ok, "expected etag to be set")
			assert.Equal(t, expected, etag, "etag did not match expected value")
		case SMaxAge:
			sMaxAge, ok := d.SMaxAge()
			assert.True(t, ok, "expected s-maxage to be set")
			assert.Equal(t, expected, sMaxAge, "s-maxage did not match expected value")
		case MustRevalidate:
			assert.Equal(t, expected, d.MustRevalidate(), "must-revalidate did not match expected value")
		case ProxyRevalidate:
			assert.Equal(t, expected, d.ProxyRevalidate(), "proxy-revalidate did not match expected value")
		case MustUnderstand:
			assert.Equal(t, expected, d.MustUnderstand(), "must-understand did not match expected value")
		case Private:
			assert.Equal(t, expected, d.Private(), "private did not match expected value")
		case Public:
			assert.Equal(t, expected, d.Public(), "public did not match expected value")
		case Immutable:
			assert.Equal(t, expected, d.Immutable(), "immutable did not match expected value")
		case StaleWhileRevalidate:
			staleWhileRevalidate, ok := d.StaleWhileRevalidate()
			assert.True(t, ok, "expected stale-while-revalidate to be set")
			assert.Equal(t, expected, staleWhileRevalidate, "stale-while-revalidate did not match expected value")
		default:
			value := d.Extension(name)
			assert.Equal(t, expected, value, "extension %q did not match expected value", name)
		}
	case *Directive:
		switch name {
		case MaxAge:
			maxAge, ok := d.MaxAge()
			assert.True(t, ok, "expected max-age to be set")
			assert.Equal(t, expected, maxAge, "max-age did not match expected value")
		case NoCache:
			assert.Equal(t, expected, d.NoCache(), "no-cache did not match expected value")
		case NoStore:
			assert.Equal(t, expected, d.NoStore(), "no-store did not match expected value")
		case NoTransform:
			assert.Equal(t, expected, d.NoTransform(), "no-transform did not match expected value")
		case StaleIfError:
			staleIfError, ok := d.StaleIfError()
			assert.True(t, ok, "expected stale-if-error to be set")
			assert.Equal(t, expected, staleIfError, "stale-if-error did not match expected value")
		default:
			value := d.Extension(name)
			assert.Equal(t, expected, value, "extension %q did not match expected value", name)
		}
	default:
		t.Fatalf("Unknown directive type: %T", directive)
	}
}

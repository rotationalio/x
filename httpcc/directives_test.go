package httpcc_test

import (
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/httpcc"
)

func TestDirective(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		directive := &httpcc.Directive{}

		date, ok := directive.Date()
		assert.False(t, ok)
		assert.True(t, date.IsZero())

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
	})

	t.Run("Populated", func(t *testing.T) {})
}

func TestRequestDirective(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		directive := &httpcc.RequestDirective{}

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

	t.Run("Populated", func(t *testing.T) {})
}

func TestResponseDirective(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		directive := &httpcc.ResponseDirective{}

		expires, ok := directive.Expires()
		assert.False(t, ok)
		assert.True(t, expires.IsZero())

		lastModified, ok := directive.LastModified()
		assert.False(t, ok)
		assert.True(t, lastModified.IsZero())

		etag, ok := directive.ETag()
		assert.False(t, ok)
		assert.Equal(t, "", etag)

		assert.False(t, directive.WeakETag())

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

	t.Run("Populated", func(t *testing.T) {})
}

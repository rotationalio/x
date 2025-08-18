package httpcc_test

import (
	"regexp"
	"testing"
	"time"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/httpcc"
)

func TestRequestBuilder(t *testing.T) {
	t.Run("MaxAge", func(t *testing.T) {
		builder := &httpcc.RequestBuilder{}
		builder.SetMaxAge(3600)
		assert.Equal(t, "max-age=3600", builder.String())
	})

	t.Run("MaxAgeZero", func(t *testing.T) {
		builder := &httpcc.RequestBuilder{}
		builder.SetMaxAge(0)
		assert.Equal(t, "max-age=0", builder.String())
	})

	t.Run("Expires", func(t *testing.T) {
		expires := time.Now().Add(2 * time.Hour)
		builder := &httpcc.RequestBuilder{}
		builder.SetExpires(expires)

		regexp := regexp.MustCompile(`^max-age=(7199|7200|7201)$`)
		assert.True(t, regexp.MatchString(builder.String()))
	})

	t.Run("Expired", func(t *testing.T) {
		expires := time.Now().Add(-2 * time.Hour)
		builder := &httpcc.RequestBuilder{}
		builder.SetExpires(expires)

		regexp := regexp.MustCompile(`^max-age=0$`)
		assert.True(t, regexp.MatchString(builder.String()))
	})

	t.Run("Empty", func(t *testing.T) {
		b := &httpcc.RequestBuilder{}
		assert.Equal(t, "", b.String())
	})

	t.Run("SingleDirective", func(t *testing.T) {
		maxAgeBuilder := &httpcc.RequestBuilder{}
		maxAgeBuilder.SetMaxAge(3600)

		testCases := []struct {
			name     string
			builder  *httpcc.RequestBuilder
			expected string
		}{
			{
				name:     "MaxAge",
				builder:  maxAgeBuilder,
				expected: "max-age=3600",
			},
			{
				name:     "MaxStale",
				builder:  &httpcc.RequestBuilder{MaxStale: 600},
				expected: "max-stale=600",
			},
			{
				name:     "MinFresh",
				builder:  &httpcc.RequestBuilder{MinFresh: 300},
				expected: "min-fresh=300",
			},
			{
				name:     "NoCache",
				builder:  &httpcc.RequestBuilder{NoCache: true},
				expected: "no-cache",
			},
			{
				name:     "NoStore",
				builder:  &httpcc.RequestBuilder{NoStore: true},
				expected: "no-store",
			},
			{
				name:     "NoTransform",
				builder:  &httpcc.RequestBuilder{NoTransform: true},
				expected: "no-transform",
			},
			{
				name:     "OnlyIfCached",
				builder:  &httpcc.RequestBuilder{OnlyIfCached: true},
				expected: "only-if-cached",
			},
			{
				name:     "Extension=",
				builder:  &httpcc.RequestBuilder{Extensions: map[string]string{"custom": "value"}},
				expected: "custom=value",
			},
			{
				name:     "Extension",
				builder:  &httpcc.RequestBuilder{Extensions: map[string]string{"custom": ""}},
				expected: "custom",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				assert.Equal(t, tc.expected, tc.builder.String())
			})
		}
	})

	t.Run("MultipleDirectives", func(t *testing.T) {
		testCases := []struct {
			b        *httpcc.RequestBuilder
			maxAge   uint64
			expected string
		}{
			{
				b:        &httpcc.RequestBuilder{NoTransform: true},
				maxAge:   3600,
				expected: "max-age=3600, no-transform",
			},
			{
				b:        &httpcc.RequestBuilder{NoCache: true, NoStore: true},
				maxAge:   0,
				expected: "no-cache, no-store",
			},
		}

		for i, tc := range testCases {
			if tc.maxAge > 0 {
				tc.b.SetMaxAge(tc.maxAge)
			}
			assert.Equal(t, tc.expected, tc.b.String(), "test case %d failed", i)
		}
	})

	t.Run("AllDirectives", func(t *testing.T) {
		b := &httpcc.RequestBuilder{
			MaxStale:     600,
			MinFresh:     300,
			NoCache:      true,
			NoStore:      true,
			NoTransform:  true,
			OnlyIfCached: true,
			Extensions: map[string]string{
				"custom": "value",
			},
		}
		b.SetMaxAge(3600)
		assert.Equal(t, "max-age=3600, max-stale=600, min-fresh=300, no-cache, no-store, no-transform, only-if-cached, custom=value", b.String())
	})
}

func TestResponseBuilder(t *testing.T) {
	t.Run("MaxAge", func(t *testing.T) {
		builder := &httpcc.ResponseBuilder{}
		builder.SetMaxAge(3600)
		assert.Equal(t, "max-age=3600", builder.String())
	})

	t.Run("MaxAgeZero", func(t *testing.T) {
		builder := &httpcc.ResponseBuilder{}
		builder.SetMaxAge(0)
		assert.Equal(t, "max-age=0", builder.String())
	})

	t.Run("Expires", func(t *testing.T) {
		expires := time.Now().Add(2 * time.Hour)
		builder := &httpcc.ResponseBuilder{}
		builder.SetExpires(expires)

		regexp := regexp.MustCompile(`^max-age=(7199|7200|7201)$`)
		assert.True(t, regexp.MatchString(builder.String()))
	})

	t.Run("Expired", func(t *testing.T) {
		expires := time.Now().Add(-2 * time.Hour)
		builder := &httpcc.ResponseBuilder{}
		builder.SetExpires(expires)

		regexp := regexp.MustCompile(`^max-age=0$`)
		assert.True(t, regexp.MatchString(builder.String()))
	})

	t.Run("SMaxAge", func(t *testing.T) {
		builder := &httpcc.ResponseBuilder{}
		builder.SetSMaxAge(1800)
		assert.Equal(t, "s-maxage=1800", builder.String())
	})

	t.Run("SMaxAgeZero", func(t *testing.T) {
		builder := &httpcc.ResponseBuilder{}
		builder.SetSMaxAge(0)
		assert.Equal(t, "s-maxage=0", builder.String())
	})

	t.Run("SExpires", func(t *testing.T) {
		expires := time.Now().Add(2 * time.Hour)
		builder := &httpcc.ResponseBuilder{}
		builder.SetSExpires(expires)

		regexp := regexp.MustCompile(`^s-maxage=(7199|7200|7201)$`)
		assert.True(t, regexp.MatchString(builder.String()))
	})

	t.Run("SExpired", func(t *testing.T) {
		expires := time.Now().Add(-2 * time.Hour)
		builder := &httpcc.ResponseBuilder{}
		builder.SetSExpires(expires)

		regexp := regexp.MustCompile(`^s-maxage=0$`)
		assert.True(t, regexp.MatchString(builder.String()))
	})

	t.Run("Empty", func(t *testing.T) {
		b := &httpcc.ResponseBuilder{}
		assert.Equal(t, "", b.String())
	})

	t.Run("SingleDirective", func(t *testing.T) {
		maxAgeBuilder := &httpcc.ResponseBuilder{}
		maxAgeBuilder.SetMaxAge(3600)

		testCases := []struct {
			name     string
			builder  *httpcc.ResponseBuilder
			expected string
		}{
			{
				name:     "MaxAge",
				builder:  maxAgeBuilder,
				expected: "max-age=3600",
			},
			{
				name:     "NoCache",
				builder:  &httpcc.ResponseBuilder{NoCache: true},
				expected: "no-cache",
			},
			{
				name:     "NoStore",
				builder:  &httpcc.ResponseBuilder{NoStore: true},
				expected: "no-store",
			},
			{
				name:     "NoTransform",
				builder:  &httpcc.ResponseBuilder{NoTransform: true},
				expected: "no-transform",
			},
			{
				name:     "Extension=",
				builder:  &httpcc.ResponseBuilder{Extensions: map[string]string{"custom": "value"}},
				expected: "custom=value",
			},
			{
				name:     "Extension",
				builder:  &httpcc.ResponseBuilder{Extensions: map[string]string{"custom": ""}},
				expected: "custom",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				assert.Equal(t, tc.expected, tc.builder.String())
			})
		}
	})

	t.Run("MultipleDirectives", func(t *testing.T) {
		testCases := []struct {
			b        *httpcc.ResponseBuilder
			maxAge   uint64
			expected string
		}{
			{
				b:        &httpcc.ResponseBuilder{NoTransform: true},
				maxAge:   3600,
				expected: "max-age=3600, no-transform",
			},
			{
				b:        &httpcc.ResponseBuilder{Private: true, Immutable: true, MustUnderstand: true},
				maxAge:   0,
				expected: "must-understand, private, immutable",
			},
		}

		for i, tc := range testCases {
			if tc.maxAge > 0 {
				tc.b.SetMaxAge(tc.maxAge)
			}
			assert.Equal(t, tc.expected, tc.b.String(), "test case %d failed", i)
		}
	})

	t.Run("AllDirectives", func(t *testing.T) {
		b := &httpcc.ResponseBuilder{
			StaleWhileRevalidate: uint64(600),
			NoCache:              true,
			NoStore:              true,
			NoTransform:          true,
			MustRevalidate:       true,
			ProxyRevalidate:      true,
			MustUnderstand:       true,
			Private:              true,
			Public:               true,
			Immutable:            true,
			Extensions: map[string]string{
				"custom": "value",
			},
		}
		b.SetMaxAge(3600)
		b.SetSMaxAge(1800)
		assert.Equal(t, "max-age=3600, s-maxage=1800, stale-while-revalidate=600, no-cache, no-store, no-transform, must-revalidate, proxy-revalidate, must-understand, private, public, immutable, custom=value", b.String())
	})
}

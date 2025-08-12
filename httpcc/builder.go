package httpcc

import (
	"strconv"
	"strings"
	"time"
)

// Allows you to build a request directive with various options.
// NOTE: performs no validation, but should be interpreted as the most restrictive
// directive takes precedence.
type RequestBuilder struct {
	maxAge       *uint64
	MaxStale     uint64
	MinFresh     uint64
	NoCache      bool
	NoStore      bool
	NoTransform  bool
	OnlyIfCached bool
	Extensions   map[string]string
}

// Set the max age in seconds for the request directive.
func (b *RequestBuilder) SetMaxAge(maxAge uint64) {
	b.maxAge = &maxAge
}

// Set the max age in seconds for the request directive from an expiration timestamp.
func (b *RequestBuilder) SetExpires(expires time.Time) {
	seconds := time.Until(expires).Seconds()

	var maxAge uint64
	if seconds < 0 {
		maxAge = 0
	} else {
		maxAge = uint64(seconds)
	}

	b.maxAge = &maxAge
}

func (b *RequestBuilder) String() string {
	started := false
	sb := strings.Builder{}

	if b.maxAge != nil {
		sb.WriteString("max-age=")
		sb.WriteString(strconv.FormatUint(*b.maxAge, 10))
		started = true
	}

	if b.MaxStale > 0 {
		if started {
			sb.WriteString(", ")
		}
		sb.WriteString("max-stale=")
		sb.WriteString(strconv.FormatUint(b.MaxStale, 10))
		started = true
	}

	if b.MinFresh > 0 {
		if started {
			sb.WriteString(", ")
		}
		sb.WriteString("min-fresh=")
		sb.WriteString(strconv.FormatUint(b.MinFresh, 10))
		started = true
	}

	if b.NoCache {
		if started {
			sb.WriteString(", ")
		}
		sb.WriteString("no-cache")
		started = true
	}

	if b.NoStore {
		if started {
			sb.WriteString(", ")
		}
		sb.WriteString("no-store")
		started = true
	}

	if b.NoTransform {
		if started {
			sb.WriteString(", ")
		}
		sb.WriteString("no-transform")
		started = true
	}

	if b.OnlyIfCached {
		if started {
			sb.WriteString(", ")
		}
		sb.WriteString("only-if-cached")
		started = true
	}

	if len(b.Extensions) > 0 {
		for key, value := range b.Extensions {
			if started {
				sb.WriteString(", ")
			}
			if key != "" {
				sb.WriteString(key)
				if value != "" {
					sb.WriteString("=")
					sb.WriteString(value)
				}
				started = true
			}
		}
	}

	return sb.String()
}

// Allows you to build a response directive with various options.
// NOTE: performs no validation, but should be interpreted as the most restrictive
// directive takes precedence.
type ResponseBuilder struct {
	maxAge               *uint64
	smaxAge              *uint64
	StaleWhileRevalidate uint64
	NoCache              bool
	NoStore              bool
	NoTransform          bool
	MustRevalidate       bool
	ProxyRevalidate      bool
	MustUnderstand       bool
	Private              bool
	Public               bool
	Immutable            bool
	Extensions           map[string]string
}

func (b *ResponseBuilder) SetMaxAge(maxAge uint64) {
	b.maxAge = &maxAge
}

// Set the max age in seconds for the request directive from an expiration timestamp.
func (b *ResponseBuilder) SetExpires(expires time.Time) {
	seconds := time.Until(expires).Seconds()

	var maxAge uint64
	if seconds < 0 {
		maxAge = 0
	} else {
		maxAge = uint64(seconds)
	}

	b.maxAge = &maxAge
}

func (b *ResponseBuilder) SetSMaxAge(sMaxAge uint64) {
	b.smaxAge = &sMaxAge
}

// Set the max age in seconds for the request directive from an expiration timestamp.
func (b *ResponseBuilder) SetSExpires(expires time.Time) {
	seconds := time.Until(expires).Seconds()

	var sMaxAge uint64
	if seconds < 0 {
		sMaxAge = 0
	} else {
		sMaxAge = uint64(seconds)
	}

	b.smaxAge = &sMaxAge
}

func (b *ResponseBuilder) String() string {
	started := false
	sb := strings.Builder{}

	if b.maxAge != nil {
		sb.WriteString("max-age=")
		sb.WriteString(strconv.FormatUint(*b.maxAge, 10))
		started = true
	}

	if b.smaxAge != nil {
		if started {
			sb.WriteString(", ")
		}
		sb.WriteString("s-maxage=")
		sb.WriteString(strconv.FormatUint(*b.smaxAge, 10))
		started = true
	}

	if b.StaleWhileRevalidate > 0 {
		if started {
			sb.WriteString(", ")
		}
		sb.WriteString("stale-while-revalidate=")
		sb.WriteString(strconv.FormatUint(b.StaleWhileRevalidate, 10))
		started = true
	}

	if b.NoCache {
		if started {
			sb.WriteString(", ")
		}
		sb.WriteString("no-cache")
		started = true
	}

	if b.NoStore {
		if started {
			sb.WriteString(", ")
		}
		sb.WriteString("no-store")
		started = true
	}

	if b.NoTransform {
		if started {
			sb.WriteString(", ")
		}
		sb.WriteString("no-transform")
		started = true
	}

	if b.MustRevalidate {
		if started {
			sb.WriteString(", ")
		}
		sb.WriteString("must-revalidate")
		started = true
	}

	if b.ProxyRevalidate {
		if started {
			sb.WriteString(", ")
		}
		sb.WriteString("proxy-revalidate")
		started = true
	}

	if b.MustUnderstand {
		if started {
			sb.WriteString(", ")
		}
		sb.WriteString("must-understand")
		started = true
	}

	if b.Private {
		if started {
			sb.WriteString(", ")
		}
		sb.WriteString("private")
		started = true
	}

	if b.Public {
		if started {
			sb.WriteString(", ")
		}
		sb.WriteString("public")
		started = true
	}

	if b.Immutable {
		if started {
			sb.WriteString(", ")
		}
		sb.WriteString("immutable")
		started = true
	}

	if len(b.Extensions) > 0 {
		for key, value := range b.Extensions {
			if started {
				sb.WriteString(", ")
			}
			if key != "" {
				sb.WriteString(key)
				if value != "" {
					sb.WriteString("=")
					sb.WriteString(value)
				}
				started = true
			}
		}
	}

	return sb.String()
}

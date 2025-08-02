package httpcc

import (
	"bufio"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// Cache-Control Directives
	MaxAge               = "max-age"
	MaxStale             = "max-stale"
	MinFresh             = "min-fresh"
	SMaxAge              = "s-maxage"
	NoCache              = "no-cache"
	NoStore              = "no-store"
	NoTransform          = "no-transform"
	OnlyIfCached         = "only-if-cached"
	MustRevalidate       = "must-revalidate"
	ProxyRevalidate      = "proxy-revalidate"
	MustUnderstand       = "must-understand"
	Private              = "private"
	Public               = "public"
	Immutable            = "immutable"
	StaleWhileRevalidate = "stale-while-revalidate"
	StaleIfError         = "stale-if-error"
)

const (
	// Header Values
	CacheControl      = "Cache-Control"
	LastModified      = "Last-Modified"
	Expires           = "Expires"
	ETag              = "ETag"
	IfNoneMatch       = "If-None-Match"
	IfModifiedSince   = "If-Modified-Since"
	IfUnmodifiedSince = "If-Unmodified-Since"
)

//===========================================================================
// Request and Response Parsing
//===========================================================================

// Request fetches the cache control directives from an HTTP request; or if a string is
// provided, it parses it as though it is  the value of a `Cache-Control` header.
func Request(req any) (directive *RequestDirective, err error) {
	switch r := req.(type) {
	case string:
		return ParseRequest(r)
	case *http.Request:
		// Parse the Cache-Control header from the request.
		if directive, err = ParseRequest(r.Header.Get(CacheControl)); err != nil {
			return nil, fmt.Errorf("failed to parse request cache control: %w", err)
		}

		// Parse the If-None-Match header if it exists.
		if ifNoneMatch := r.Header.Get(IfNoneMatch); ifNoneMatch != "" {
			directive.ifNoneMatch = &ifNoneMatch
		}

		// Parse the If-Unmodified-Since header if it exists.
		if ifUnmodifiedSince := r.Header.Get(IfUnmodifiedSince); ifUnmodifiedSince != "" {
			var t time.Time
			if t, err = http.ParseTime(ifUnmodifiedSince); err != nil {
				return nil, fmt.Errorf("failed to parse if-unmodified-since time: %w", err)
			}
			directive.ifUnmodifiedSince = &t
		}

		// Parse the If-Modified-Since header if it exists.
		if ifModifiedSince := r.Header.Get(IfModifiedSince); ifModifiedSince != "" {
			var t time.Time
			if t, err = http.ParseTime(ifModifiedSince); err != nil {
				return nil, fmt.Errorf("failed to parse if-modified-since time: %w", err)
			}
			directive.ifModifiedSince = &t
		}
		return directive, nil
	case http.Request:
		return Request(&r)
	case *RequestDirective:
		return r, nil
	default:
		return nil, fmt.Errorf("unsupported type %T for request cache control parsing", req)
	}
}

// Response fetches the cache control directives from an HTTP response; or if a string
// is provided, it parses it as though it is the value of a `Cache-Control` header.
func Response(rep any) (directive *ResponseDirective, err error) {
	switch r := rep.(type) {
	case string:
		return ParseResponse(r)
	case *http.Response:
		// Parse the Cache-Control header from the response.
		if directive, err = ParseResponse(r.Header.Get(CacheControl)); err != nil {
			return nil, fmt.Errorf("failed to parse response cache control: %w", err)
		}

		// Parse the Expires header if it exists.
		if expires := r.Header.Get(Expires); expires != "" {
			var t time.Time
			if t, err = http.ParseTime(expires); err != nil {
				return nil, fmt.Errorf("failed to parse expires time: %w", err)
			}
			directive.expires = &t
		}

		// Parse the Last-Modified header if it exists.
		if lastModified := r.Header.Get(LastModified); lastModified != "" {
			var t time.Time
			if t, err = http.ParseTime(lastModified); err != nil {
				return nil, fmt.Errorf("failed to parse last-modified time: %w", err)
			}
			directive.lastModified = &t
		}

		// Parse the ETag header if it exists.
		if etag := r.Header.Get(ETag); etag != "" {
			directive.etag = &etag
		}
		return directive, nil
	case http.Response:
		return Response(&r)
	case *ResponseDirective:
		return r, nil
	default:
		return nil, fmt.Errorf("unsupported type %T for response cache control parsing", rep)
	}
}

//===========================================================================
// Cache-Control Directive Parsing
//===========================================================================

// ParseRequest converts a request cache control header string into a RequestDirective.
func ParseRequest(cc string) (_ *RequestDirective, err error) {
	var (
		dir    RequestDirective
		tokens []*TokenPair
	)

	dir.extensions = make(map[string]string)
	if tokens, err = ParseRequestDirectives(cc); err != nil {
		return nil, fmt.Errorf("failed to parse tokens: %w", err)
	}

	for _, token := range tokens {
		name := strings.ToLower(token.Name)
		switch name {
		case MaxAge:
			var val uint64
			if val, err = parseUint64(token.Value); err != nil {
				return nil, fmt.Errorf("could not parse max-age: %w", err)
			}
			dir.maxAge = &val
		case MaxStale:
			var val uint64
			if val, err = parseUint64(token.Value); err != nil {
				return nil, fmt.Errorf("could not parse max-stale: %w", err)
			}
			dir.maxStale = &val
		case MinFresh:
			var val uint64
			if val, err = parseUint64(token.Value); err != nil {
				return nil, fmt.Errorf("could not parse min-fresh: %w", err)
			}
			dir.minFresh = &val
		case StaleIfError:
			var val uint64
			if val, err = parseUint64(token.Value); err != nil {
				return nil, fmt.Errorf("could not parse stale-if-error: %w", err)
			}
			dir.staleIfError = &val
		case NoCache:
			dir.noCache = true
		case NoStore:
			dir.noStore = true
		case NoTransform:
			dir.noTransform = true
		case OnlyIfCached:
			dir.onlyIfCached = true
		default:
			if token.Value == "" {
				token.Value = token.Name
			}
			dir.extensions[name] = token.Value
		}
	}

	return &dir, nil
}

// ParseResponse converts a response cache control header string into a ResponseDirective.
func ParseResponse(cc string) (_ *ResponseDirective, err error) {
	var (
		dir    ResponseDirective
		tokens []*TokenPair
	)

	dir.extensions = make(map[string]string)
	if tokens, err = ParseResponseDirectives(cc); err != nil {
		return nil, fmt.Errorf("failed to parse tokens: %w", err)
	}

	for _, token := range tokens {
		name := strings.ToLower(token.Name)
		switch name {
		case MaxAge:
			var val uint64
			if val, err = parseUint64(token.Value); err != nil {
				return nil, fmt.Errorf("could not parse max-age: %w", err)
			}
			dir.maxAge = &val
		case SMaxAge:
			var val uint64
			if val, err = parseUint64(token.Value); err != nil {
				return nil, fmt.Errorf("could not parse s-maxage: %w", err)
			}
			dir.sMaxAge = &val
		case StaleWhileRevalidate:
			var val uint64
			if val, err = parseUint64(token.Value); err != nil {
				return nil, fmt.Errorf("could not parse stale-while-revalidate: %w", err)
			}
			dir.staleWhileRevalidate = &val
		case StaleIfError:
			var val uint64
			if val, err = parseUint64(token.Value); err != nil {
				return nil, fmt.Errorf("could not parse stale-if-error: %w", err)
			}
			dir.staleIfError = &val
		case NoCache:
			dir.noCache = true
		case MustRevalidate:
			dir.mustRevalidate = true
		case ProxyRevalidate:
			dir.proxyRevalidate = true
		case NoStore:
			dir.noStore = true
		case Private:
			dir.private = true
		case Public:
			dir.public = true
		case MustUnderstand:
			dir.mustUnderstand = true
		case NoTransform:
			dir.noTransform = true
		case Immutable:
			dir.immutable = true
		default:
			if token.Value == "" {
				token.Value = token.Name
			}
			dir.extensions[name] = token.Value
		}
	}

	return &dir, nil
}

//===========================================================================
// Directive Parser
//===========================================================================

type TokenPair struct {
	Name  string
	Value string
}

type TokenValuePolicy uint8

const (
	NoArgument TokenValuePolicy = iota
	TokenOnly
	QuotedStringOnly
	AnyTokenValue
)

type directiveValidator func(string) TokenValuePolicy

// ParseRequestDirective parses a single token with validation for request directives.
func ParseRequestDirective(s string) (*TokenPair, error) {
	return parseDirective(s, func(name string) TokenValuePolicy {
		switch name {
		case MaxAge, MaxStale, MinFresh, StaleIfError:
			return TokenOnly
		case NoCache, NoStore, NoTransform, OnlyIfCached:
			return NoArgument
		default:
			return AnyTokenValue
		}
	})
}

// ParseResponseDirective parses a single token with validation for response directives.
func ParseResponseDirective(s string) (*TokenPair, error) {
	return parseDirective(s, func(name string) TokenValuePolicy {
		switch name {
		case MaxAge, SMaxAge, StaleWhileRevalidate, StaleIfError:
			return TokenOnly
		case NoCache, MustRevalidate, ProxyRevalidate, NoStore, Private, Public, MustUnderstand, NoTransform, Immutable:
			return NoArgument
		default:
			return AnyTokenValue
		}
	})
}

// Handles a single token directive using the specified validation function to determine
// what the token pair policy is and how to construct the TokenPair.
func parseDirective(s string, ccd directiveValidator) (pair *TokenPair, err error) {
	s = strings.TrimSpace(s)
	i := strings.IndexByte(s, '=')
	if i == -1 {
		return &TokenPair{Name: s, Value: ""}, nil
	}

	pair = &TokenPair{Name: strings.TrimSpace(s[:i])}
	v := strings.TrimSpace(s[i+1:])
	if len(v) == 0 {
		// `key=` should be a parse error but it's HTTP so we return as if nothing happened.
		return pair, nil
	}

	switch ccd(pair.Name) {
	case TokenOnly:
		if v[0] == '"' {
			return nil, fmt.Errorf(`invalid value for %s (quoted string not allowed)`, pair.Name)
		}
	case QuotedStringOnly:
		if v[0] != '"' {
			return nil, fmt.Errorf(`invalid value for %s (expected quoted string)`, pair.Name)
		}

		var tmp string
		if tmp, err = strconv.Unquote(v); err != nil {
			return nil, fmt.Errorf(`invalid value for %s (unquoting failed: %w)`, pair.Name, err)
		}
		v = tmp
	case AnyTokenValue:
		if v[0] == '"' {
			var tmp string
			if tmp, err = strconv.Unquote(v); err != nil {
				return nil, fmt.Errorf(`invalid value for %s (unquoting failed: %w)`, pair.Name, err)
			}
			v = tmp
		}
	case NoArgument:
		if len(v) > 0 {
			return nil, fmt.Errorf(`invalid value for %s (no argument expected)`, pair.Name)
		}
	}

	pair.Value = v
	return pair, nil
}

// Parses all the directives in a request cache control header string.
func ParseRequestDirectives(cc string) (tokens []*TokenPair, err error) {
	return parseDirectives(cc, ParseRequestDirective)
}

// Parses all the directives in a response cache control header string.
func ParseResponseDirectives(cc string) (tokens []*TokenPair, err error) {
	return parseDirectives(cc, ParseResponseDirective)
}

// Parses the string into token pairs based on the policies and validation provided
// by the handler parsing function p.
func parseDirectives(s string, p func(string) (*TokenPair, error)) (tokens []*TokenPair, err error) {
	scanner := bufio.NewScanner(strings.NewReader(s))
	scanner.Split(commaSeparatedWords)

	for scanner.Scan() {
		var token *TokenPair
		if token, err = p(scanner.Text()); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return tokens, nil
}

// Scan to find comma separated tokens, handling leading spaces and consecutive spaces.
func commaSeparatedWords(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// Skip leading spaces.
	start := 0
	for width := 0; start < len(data); start += width {
		var r rune
		r, width = utf8.DecodeRune(data[start:])
		if !isSpace(r) {
			break
		}
	}

	// Scan until we find a comma. Keep track of consecutive whitespaces.
	var ws int
	for width, i := 0, start; i < len(data); i += width {
		var r rune
		r, width = utf8.DecodeRune(data[i:])
		switch {
		case isSpace(r):
			ws++
		case r == ',':
			return i + width, data[start : i-ws], nil
		default:
			ws = 0
		}
	}

	// If we're at EOF, we have a final, non-empty, non-terminated word. Return it.
	if atEOF && len(data) > start {
		return len(data), data[start : len(data)-ws], nil
	}

	// Request more data.
	return start, nil, nil
}

// Checks if the rune is a space character according to the HTTP spec.
func isSpace(r rune) bool {
	if r <= '\u00FF' {
		// Obvious ASCII ones: \t through \r plus space. Plus some Latin-1 spaces.
		switch r {
		case ' ', '\t', '\n', '\v', '\f', '\r':
			return true
		case '\u0085', '\u00A0':
			return true
		default:
			return false
		}
	}

	// High-valued spaces.
	if '\u2000' <= r && r <= '\u200A' {
		return true
	}

	switch r {
	case '\u1680', '\u2028', '\u2029', '\u202F', '\u205F', '\u3000':
		return true
	}

	return false
}

func parseUint64(v string) (i uint64, err error) {
	if v == "" {
		return 0, nil // Default value for empty strings.
	}

	if i, err = strconv.ParseUint(v, 10, 64); err != nil {
		return 0, err
	}
	return i, nil
}

package httpcc

import "time"

// Directive includes cache directives for both requests and responses.
type Directive struct {
	maxAge       *uint64
	noCache      bool
	noStore      bool
	noTransform  bool
	staleIfError *uint64
	extensions   map[string]string
}

// Directives that are only part of an HTTP request. These directives are usually
// parsed on the server side and constructed by a client to send to the server.
type RequestDirective struct {
	Directive
	ifNoneMatch       *string
	ifUnmodifiedSince *time.Time
	ifModifiedSince   *time.Time
	maxStale          *uint64
	minFresh          *uint64
	onlyIfCached      bool
}

// Directives that a server sends back to a client in an HTTP response. These directives
// are used to control caching behavior on the client side or in intermediate caches.
type ResponseDirective struct {
	Directive
	expires              *time.Time
	lastModified         *time.Time
	etag                 *string
	weakEtag             bool
	sMaxAge              *uint64
	mustRevalidate       bool
	proxyRevalidate      bool
	mustUnderstand       bool
	private              bool
	public               bool
	immutable            bool
	staleWhileRevalidate *uint64
}

//===========================================================================
// Directive Methods
//===========================================================================

// The max-age=N response directive indicates that the response remains fresh until
// N seconds after the response is generated. Stored HTTP responses have two states:
// fresh and stale. The fresh state usually indicates that the response is still valid
// and can be reused, while the stale state means that the cached response has already
// expired. The criterion for determining when a response is fresh and when it is stale
// is age. In HTTP, age is the time elapsed since the response was generated.
//
// For requests, The max-age=N request directive indicates that the client allows a
// stored response that is generated on the origin server within N seconds — where N
// may be any non-negative integer (including 0).
func (d *Directive) MaxAge() (uint64, bool) {
	if v := d.maxAge; v != nil {
		return *v, true
	}
	return 0, false
}

// The no-cache response directive indicates that the response can be stored in caches,
// but the response must be validated with the origin server before each reuse, even
// when the cache is disconnected from the origin server.
//
// For requests, the no-cache request directive asks caches to validate the response
// with the origin server before reuse.
func (d *Directive) NoCache() bool {
	return d.noCache
}

// The no-store response directive indicates that any caches of any kind (private or
// shared) should not store this response.
//
// For requests, the no-store request directive allows a client to request that caches
// refrain from storing the request and corresponding response — even if the origin
// server's response could be stored.
func (d *Directive) NoStore() bool {
	return d.noStore
}

// Some intermediaries transform content for various reasons. For example, some convert
// images to reduce transfer size. In some cases, this is undesirable for the content
// provider. no-transform indicates that any intermediary (regardless of whether it
// implements a cache) shouldn't transform the response contents.
func (d *Directive) NoTransform() bool {
	return d.noTransform
}

// The stale-if-error request directive indicates that the browser is interested in
// receiving stale content on error from any intermediate server for a particular
// origin. This is not supported by any browser.
func (d *Directive) StaleIfError() (uint64, bool) {
	if v := d.staleIfError; v != nil {
		return *v, true
	}
	return 0, false
}

// Extensions are used to collect additional information stored in the directive that
// are not part of the standard HTTP cache directives. These extensions can be used
// to provide additional metadata or custom behavior for caching.
func (d *Directive) Extensions() map[string]string {
	return d.extensions
}

// Get a specific extension value by its key. If the key does not exist, it returns an
// empty string. Note that if a boolean value is stored as an extension, it will return
// the name of the extension as a string if it has been set.
func (d *Directive) Extension(s string) string {
	return d.extensions[s]
}

//===========================================================================
// Request Directive Methods
//===========================================================================

// The HTTP If-None-Match request header makes a request conditional. The server returns
// the requested resource in GET and HEAD methods with a 200 status, only if it doesn't
// have an ETag matching the ones in the If-None-Match header.
// See: https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/If-None-Match
func (d *RequestDirective) IfNoneMatch() (string, bool) {
	if v := d.ifNoneMatch; v != nil {
		return *v, true
	}
	return "", false
}

// The HTTP If-Unmodified-Since request header makes the request for the resource
// conditional. The server will send the requested resource (or accept it in the case
// of a POST or another non-safe method) only if the resource on the server has not
// been modified after the date in the request header.
// See: https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/If-Unmodified-Since
func (d *RequestDirective) IfUnmodifiedSince() (time.Time, bool) {
	if v := d.ifUnmodifiedSince; v != nil {
		return *v, true
	}
	return time.Time{}, false
}

// The HTTP If-Modified-Since request header makes a request conditional. The server
// sends back the requested resource, with a 200 status, only if it has been modified
// after the date in the If-Modified-Since header. If the resource has not been modified
// since, the response is a 304 without any body, and the Last-Modified response header
// of the previous request contains the date of the last modification.
// See: https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/If-Modified-Since
func (d *RequestDirective) IfModifiedSince() (time.Time, bool) {
	if v := d.ifModifiedSince; v != nil {
		return *v, true
	}
	return time.Time{}, false
}

// The max-stale=N request directive indicates that the client allows a stored response
// that is stale within N seconds. If no N value is specified, the client will accept a
// stale response of any age.
func (d *RequestDirective) MaxStale() (uint64, bool) {
	if v := d.maxStale; v != nil {
		return *v, true
	}
	return 0, false
}

// The min-fresh=N request directive indicates that the client allows a stored
// response that is fresh for at least N seconds.
func (d *RequestDirective) MinFresh() (uint64, bool) {
	if v := d.minFresh; v != nil {
		return *v, true
	}
	return 0, false
}

// The client indicates that an already-cached response should be returned. If a cache
// has a stored response, even a stale one, it will be returned. If no cached response
// is available, a 504 Gateway Timeout response will be returned.
func (d *RequestDirective) OnlyIfCached() bool {
	return d.onlyIfCached
}

//===========================================================================
// Response Directive Methods
//===========================================================================

// The HTTP Expires response header contains the date/time after which the response is
// considered expired in the context of HTTP caching.
// See: https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Expires
func (d *ResponseDirective) Expires() (time.Time, bool) {
	if v := d.expires; v != nil {
		return *v, true
	}
	return time.Time{}, false
}

// The HTTP Last-Modified response header contains a date and time when the origin
// server believes the resource was last modified. It is used as a validator in
// conditional requests (If-Modified-Since or If-Unmodified-Since) to determine if a
// requested resource is the same as one already stored by the client. It is less
// accurate than an ETag for determining file contents, but can be used as a fallback
// mechanism if ETags are unavailable.
//
// See: https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Last-Modified
func (d *ResponseDirective) LastModified() (time.Time, bool) {
	if v := d.lastModified; v != nil {
		return *v, true
	}
	return time.Time{}, false
}

// The HTTP ETag (entity tag) response header is an identifier for a specific version
// of a resource. It lets caches be more efficient and save bandwidth, as a web server
// does not need to resend a full response if the content has not changed.
// See: https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/ETag
func (d *ResponseDirective) ETag() (string, bool) {
	if v := d.etag; v != nil {
		return *v, true
	}
	return "", false
}

// Weak ETags are a variant of Etags that are faster to compute but may have collisions.
// If a weak etag was sent in the response, this method returns true.
func (d *ResponseDirective) WeakETag() bool {
	return d.weakEtag
}

// The s-maxage response directive indicates how long the response remains fresh in a
// shared cache. The s-maxage directive is ignored by private caches, and overrides the
// value specified by the max-age directive or the Expires header for shared caches,
// if they are present.
func (d *ResponseDirective) SMaxAge() (uint64, bool) {
	if v := d.sMaxAge; v != nil {
		return *v, true
	}
	return 0, false
}

// The must-revalidate response directive indicates that the response can be stored in
// caches and can be reused while fresh. If the response becomes stale, it must be
// validated with the origin server before reuse.
func (d *ResponseDirective) MustRevalidate() bool {
	return d.mustRevalidate
}

// The proxy-revalidate response directive is the equivalent of must-revalidate,
// but specifically for shared caches only.
func (d *ResponseDirective) ProxyRevalidate() bool {
	return d.proxyRevalidate
}

// The private response directive indicates that the response can be stored only in a
// private cache (e.g., local caches in browsers).
func (d *ResponseDirective) Private() bool {
	return d.private
}

// The public response directive indicates that the response can be stored in a shared
// cache. Responses for requests with Authorization header fields must not be stored in
// a shared cache; however, the public directive will cause such responses to be stored
// in a shared cache.
func (d *ResponseDirective) Public() bool {
	return d.public
}

// The must-understand response directive indicates that a cache should store the
// response only if it understands the requirements for caching based on status code.
// must-understand should be coupled with no-store for fallback behavior.
func (d *ResponseDirective) MustUnderstand() bool {
	return d.mustUnderstand
}

// The immutable response directive indicates that the response will not
// be updated while it's fresh.
func (d *ResponseDirective) Immutable() bool {
	return d.immutable
}

// The stale-while-revalidate response directive indicates that the cache could reuse
// a stale response while it revalidates it to a cache.
func (d *ResponseDirective) StaleWhileRevalidate() (uint64, bool) {
	if v := d.staleWhileRevalidate; v != nil {
		return *v, true
	}
	return 0, false
}

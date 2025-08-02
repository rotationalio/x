# Cache Control (httpcc)

Implements [RFC 7234](http://tools.ietf.org/html/rfc7234) Hypertext Transfer Protocol (HTTP/1.1): Caching. It does this by parsing the Cache-Control and other headers, providing cache directives about requests and responses. Note `httpcc` does not implement an actual cache backend, just the directives to influence cache behavior.

## Parsing Cache Control Headers

The [cache control directives](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Cache-Control) are specified in the `Cache-Control` headers of both HTTP requests and responses (and the available directives are slightly different for each).

To parse request directives:

```go
directives, err := httpcc.Request(req)
```

And to parse response directives:

```go
directives, err := httpcc.Response(rep)
```

You can also pass in a cache control string using `req.Headers.Get("Cache-Control")` to
parse that header by itself, though other headers will not be available in the response.

## Other Cache Control Headers

This package also supports other headers that may influence caching such as the following response headers:

- `Expires`
- `Last-Modified`
- `ETag`

And the following request headers:

- `If-None-Match`
- `If-Modified-Since`
- `If-Unmodified-Since`

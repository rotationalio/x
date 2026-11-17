# Hawk

Authentication mechanism that implements an authorization header to authenticate webhook requests using a similar shared secret mechanism as AWS4-HMAC-SHA256 as defined here by [AWS sigv4](https://docs.aws.amazon.com/AmazonS3/latest/API/sigv4-auth-using-authorization-header.html). This is also very similar to the [Mozilla Hawk](https://github.com/mozilla/hawk) authentication scheme, but since this authentication is archived and no longer maintained, we're moving forward with the AWS Signature Version 4 mechanism.

This method is best for machine-to-machine API keys (especially license keys) that have an client ID and shared secret. The shared secret is used to create an HMAC signature that verifies that the request is legitimate without transmitting the secret across the wire. This mechanism can also be used to prove that a message has not been tampered with in flight.

## Usage

```go
import "go.rtnl.ai/x/hawk"
```

Both sides need the key ID, which is sent in the token to identify the client, and the shared secret (a 32 byte key), which is never sent. The examples below sign two headers, `X-Transfer-ID` and `X-Transfer-Timestamp`, on both the request and the reply.

### Signing a Request (Client)

Set the headers on the request, then create an `HMAC` with `New` and `Append` each header to be signed. `Authorization` returns the value of the `Authorization` header:

```go
req, err := http.NewRequest(http.MethodPost, "https://example.com/v1/transfers", body)
if err != nil {
    return err
}

req.Header.Set("X-Transfer-ID", transferID)
req.Header.Set("X-Transfer-Timestamp", time.Now().UTC().Format(time.RFC3339))

mac := hawk.New(keyID, secret)
mac.Append("X-Transfer-ID", req.Header.Get("X-Transfer-ID"))
mac.Append("X-Transfer-Timestamp", req.Header.Get("X-Transfer-Timestamp"))

auth, err := mac.Authorization()
if err != nil {
    return err
}

req.Header.Set("Authorization", auth)
```

The header contains the signature, a random nonce, the names of the signed headers, and the key ID:

```
Authorization: HMAC sig=A5CciwEllyejQhOIzrOhRZKSDQkiE01pRbfkzlV9EO8, nonce=lV-8iRl3DKn8GFSniLvsBA, headers=x-transfer-id;x-transfer-timestamp, kid=01JT4B3R5Z6AHJXV87QHPPKRBM
```

Create a new `HMAC` for every request so that each one is signed with its own nonce. The signature covers the name and the value of every header in the order they were appended (see [Signed Data](#signed-data)), so the values on the request must be exactly the values that were appended.

To sign every request that an `http.Client` sends without doing this by hand, use a [Transport](#transport).

### Verifying a Request (Server)

`Parse` the `Authorization` header into a `Token`, look up the secret for the token's `KeyID`, `Collect` the signed header values from the request, and `Verify` the signature:

```go
func Transfer(w http.ResponseWriter, r *http.Request) {
    token, err := hawk.Parse(r.Header.Get("Authorization"))
    if err != nil {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    // Fetch the shared secret that belongs to the key ID from your key store.
    secret, err := lookupSecret(token.KeyID())
    if err != nil {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    // Collect the required headers and verify the signature.
    token.Collect(r.Header)
    if ok, err := token.Verify(secret); err != nil || !ok {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    // The request is authentic: handle it and sign the reply (see below).
}
```

`Collect` reads the headers that are named in the token from the request. Only the first value of each header is collected: if a header is repeated on the request, the additional values are not covered by the signature, so read signed headers with `Header.Get` and not `Header.Values`. A header that is named in the token but missing from the request is collected as an empty value, which fails verification unless an empty value was signed.

The package verifies signatures; it does not detect replays. Reject requests whose `X-Transfer-Timestamp` is outside of the window you are willing to accept and requests whose `X-Transfer-ID` has already been handled.

### Signing the Reply (Server)

The server proves that it also holds the secret by signing its response headers the same way and sending the token in the `Server-Authorization` header. At the end of the handler above:

```go
    w.Header().Set("X-Transfer-ID", r.Header.Get("X-Transfer-ID"))
    w.Header().Set("X-Transfer-Timestamp", time.Now().UTC().Format(time.RFC3339))

    mac := hawk.New(token.KeyID(), secret)
    mac.Append("X-Transfer-ID", w.Header().Get("X-Transfer-ID"))
    mac.Append("X-Transfer-Timestamp", w.Header().Get("X-Transfer-Timestamp"))

    auth, err := mac.Authorization()
    if err != nil {
        http.Error(w, "internal server error", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Server-Authorization", auth)
    w.WriteHeader(http.StatusNoContent)
```

The headers must be set before the status or the body is written.

### Verifying the Reply (Client)

The client verifies the `Server-Authorization` header exactly as the server verified the request, using the secret it signed the request with:

```go
rep, err := http.DefaultClient.Do(req)
if err != nil {
    return err
}
defer rep.Body.Close()

token, err := hawk.Parse(rep.Header.Get("Server-Authorization"))
if err != nil {
    return err
}

if token.KeyID() != keyID {
    return errors.New("reply signed with an unknown key")
}

for _, header := range []string{"x-transfer-id", "x-transfer-timestamp"} {
    if !slices.Contains(token.Headers(), header) || rep.Header.Get(header) == "" {
        return errors.New("reply does not sign the required headers")
    }
}

token.Collect(rep.Header)
if ok, err := token.Verify(secret); err != nil || !ok {
    return errors.New("could not verify the reply")
}

if rep.Header.Get("X-Transfer-ID") != transferID {
    return errors.New("reply is for a different transfer")
}
```

### Credentials

A `Credentialing` object supplies the key ID and the key that a request is signed with. The package has two implementations for a client that has one static set of credentials. You can also implement your own `Credentialing` interface to read the credentials from a database and/or to fetch the KeyID from the request or the context on the request.

A `Credential` is a static single credential that holds the client ID and the secret as separate fields, and an `APIKey` is the same two values in one string, separated by a colon:

```go
creds := &hawk.Credential{
    ClientID: "01JT4B3R5Z6AHJXV87QHPPKRBM",
    Secret:   "z7q8RxW0dZ1FuiaVPdL8C/wjRO9wogBUMufxa1CBYQ0",
}

apiKey := hawk.APIKey("01JT4B3R5Z6AHJXV87QHPPKRBM:z7q8RxW0dZ1FuiaVPdL8C/wjRO9wogBUMufxa1CBYQ0")
```

The secret is the key encoded as base64 with the standard alphabet and no padding (`base64.RawStdEncoding`), and the key must be at least 32 bytes. Note that this is not the URL safe encoding that the token uses for the signature and the nonce. To create a secret:

```go
key := make([]byte, 32)
if _, err := rand.Read(key); err != nil {
    return err
}

secret := base64.RawStdEncoding.EncodeToString(key)
```

An `APIKey` is split on the first colon, so the client ID cannot contain one. A `Credential` has `client_id` and `client_secret` tags for JSON, YAML, and msgpack, and `env` tags that name `HAWK_CLIENT_ID` and `HAWK_CLIENT_SECRET` for a configuration loader that reads them; the package does not read the environment itself.

Both are validated when the key is requested, which is when a request is signed and not when the credentials are created:

| Error | Cause |
| --- | --- |
| `ErrMissingKeyID` | The client ID is empty. |
| `ErrInvalidKeyID` | The key was requested for a key ID that is not the client ID. |
| `ErrInvalidSecret` | The API key has no colon, or the key is shorter than 32 bytes. |
| `base64.CorruptInputError` | The secret is not base64, is padded, or uses the URL safe alphabet. |

The server verifies a token with the key, not with the encoded secret. A server that stores the same credentials can decode the key with them, which also checks that the token was signed with the expected key ID:

```go
key, err := creds.Key(token.KeyID())
if err != nil {
    http.Error(w, "unauthorized", http.StatusUnauthorized)
    return
}

token.Collect(r.Header)
if ok, err := token.Verify(key); err != nil || !ok {
    http.Error(w, "unauthorized", http.StatusUnauthorized)
    return
}
```

Implement `Credentialing` to supply credentials any other way, for example to fetch the secret from a vault or to choose the credentials by request. `KeyID` is called with the request that is about to be signed and `Key` is then called with the key ID that it returned:

```go
// Signs a request with the API key of the host that the request is sent to.
type HostKeys map[string]hawk.APIKey

func (h HostKeys) KeyID(req *http.Request) string {
    return h[req.URL.Host].KeyID(req)
}

func (h HostKeys) Key(keyID string) ([]byte, error) {
    for _, apiKey := range h {
        if apiKey.KeyID(nil) == keyID {
            return apiKey.Key(keyID)
        }
    }
    return nil, hawk.ErrMissingKeyID
}
```

An implementation must be safe for concurrent use, must not modify the request, and must return an error from `Key` if it has no key so that the request is not sent.

### Transport

A `Transport` is an `http.RoundTripper` that signs every request with the key from its `Credentials` and sets the `Authorization` header, so that requests do not have to be signed individually:

```go
client := &http.Client{
    Transport: &hawk.Transport{Credentials: creds},
}

rep, err := client.Post("https://example.com/v1/transfers", "application/json", body)
```

By default the transport signs the `X-Request-ID` and `X-Timestamp` headers. If the request does not have them, the transport sets the request ID to a new UUID and the timestamp to the current time in RFC 3339 format; values that are already on the request are signed as they are:

```
X-Request-ID: 0576d5e6-ac50-41e6-973d-55b5c1d678c3
X-Timestamp: 2026-11-17T16:59:54-06:00
Authorization: HMAC sig=1BN0yKHQ6uDRnQH5F-CcXPkxgjPXAeWlyFT8V3etOHg, nonce=XcB9zbSnpcip3lAR_qPgpA, headers=x-request-id;x-timestamp, kid=01JT4B3R5Z6AHJXV87QHPPKRBM
```

The transport has three fields:

- `Credentials` is required; a transport without credentials panics when it is used.
- `Headers` are the names of the headers to sign, in order, instead of the defaults.
- `Next` is the round tripper that sends the signed request, which is `http.DefaultTransport` if it is not set. Use it to configure the connection or to chain other middleware.

When `Headers` is set, the transport only signs: it does not add `X-Request-ID` or `X-Timestamp`, and it does not set a value for any header. Set the headers on the request before it is sent, since a header that is missing is signed with an empty value:

```go
client := &http.Client{
    Transport: &hawk.Transport{
        Credentials: creds,
        Headers:     []string{"X-Transfer-ID", "X-Transfer-Timestamp"},
        Next:        &http.Transport{TLSHandshakeTimeout: 5 * time.Second},
    },
}

req, err := http.NewRequest(http.MethodPost, "https://example.com/v1/transfers", body)
if err != nil {
    return err
}

req.Header.Set("X-Transfer-ID", transferID)
req.Header.Set("X-Transfer-Timestamp", time.Now().UTC().Format(time.RFC3339))

rep, err := client.Do(req)
```

Some things to be aware of when using the transport:

- The request that is passed to the client is not modified; the headers are added to a copy that is sent. To know the request ID, for example to log it, set `X-Request-ID` on the request instead of having the transport generate it.
- An `Authorization` header that is already on the request is replaced on the request that is sent.
- Every round trip is signed with a new nonce, so a request that is sent again gets a new token, and a new request ID and timestamp if they are generated by the transport.
- If the credentials return an error then the request is not sent. The client wraps the error in a `*url.Error`, so test for it with `errors.Is(err, hawk.ErrInvalidSecret)`.
- Only the listed headers are signed; the URL and the body of the request are not.
- The transport does not verify the `Server-Authorization` header of the reply; see [Verifying the Reply](#verifying-the-reply-client).

The server verifies a request from the transport as described in [Verifying a Request](#verifying-a-request-server). The client chooses which headers are signed, so the server must also check that `token.Headers()` contains every header that it relies on, such as `x-request-id` and `x-timestamp`, before it uses them to reject replays.

### Signed Data

The signature is the HMAC-SHA256 of the nonce followed by one `name:value\n` line for each signed header, in the order that the headers are listed in the token:

```
nonce || "x-transfer-id:f17c9693-c280-4836-b544-245d832a11e0\n" || "x-transfer-timestamp:2026-11-17T20:15:00Z\n"
```

- The nonce is the 16 raw bytes, not the base64 encoding that is sent in the token.
- Each line is the lowercase header name, a colon, the header value, and a newline, with nothing added around the colon.
- The value is signed exactly as it is sent and is not normalized. HTTP strips leading and trailing whitespace from header values in transit, so do not sign a value that has any.
- Only the first value of a header is signed; additional values of a repeated header are ignored.
- A header name cannot contain a colon and a value cannot contain a newline. `Parse` rejects a token that names a header with a colon.
- The signature and the nonce are base64 URL encoded without padding in the token.

Since the header names are part of the signed data, renaming, removing, or reordering the headers in the token fails verification.
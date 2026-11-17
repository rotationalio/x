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
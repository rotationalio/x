package hawk

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
)

var (
	ErrInvalidHMACToken = errors.New("invalid authorization hmac token")
)

// HMAC implements an authorization header to authenticate webhook requests using a
// similar shared secret mechanism as AWS4-HMAC-SHA256 as defined here:
// https://docs.aws.amazon.com/AmazonS3/latest/API/sigv4-auth-using-authorization-header.html
//
// Create an HMAC object with a 32 byte key and a string that represents the key ID.
// Headers then must be appended to the HMAC in order to generate the data to sign.
// Once signed, an HMAC-SHA256 signature is generated and can be used to create the
// Authorization header.
type HMAC struct {
	headers []string
	data    []byte
	key     []byte
	keyID   string
}

// Creates a new HMAC object with the given key ID and key (also referred to as the
// client ID and shared or client secret). The HMAC is generally safe for one time use,
// and a new HMAC object should be created for each request.
func New(keyID string, key []byte) *HMAC {
	hmac := &HMAC{
		headers: make([]string, 0),
		data:    make([]byte, 16, 1024),
		key:     key,
		keyID:   keyID,
	}

	// Create a nonce to prevent reply attacks.
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		panic(fmt.Errorf("could not create nonce for HMAC: %w", err))
	}

	copy(hmac.data[:16], nonce[:])
	return hmac
}

// Add the header and the header value to the HMAC object. Note that the order that
// the headers are added is important as each header is appended to the data to be
// signed as a canonical "name:value\n" line, where the name is lowercase and the value
// is not modified. If the header is already present, it will not be added again.
// Headers are stored as lowercase values.
//
// The header name must not contain a colon and the value must not contain a newline;
// neither is valid in an HTTP header and the lines would no longer be unambiguous.
func (h *HMAC) Append(header, value string) {
	header = strings.ToLower(header)
	if slices.Contains(h.headers, header) {
		return
	}

	h.headers = append(h.headers, header)
	h.data = canonical(h.data, header, value)
}

// Appends the canonical "name:value\n" line for the header to the data to be signed.
// Both the signer and the verifier must construct the data with this function.
func canonical(data []byte, header, value string) []byte {
	data = append(data, header...)
	data = append(data, ':')
	data = append(data, value...)
	return append(data, '\n')
}

// Nonce returns the base64 encoded nonce that is used to prevent replay attacks.
func (h *HMAC) Nonce() string {
	return base64.RawURLEncoding.EncodeToString(h.data[:16])
}

// Signature returns the base64 encoded HMAC-SHA256 signature of the data.
func (h *HMAC) Signature() (_ string, err error) {
	mac := hmac.New(sha256.New, h.key)
	if _, err = mac.Write(h.data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

// Headers returns the semicolon list of request headers used to compute the signature.
// The list includes header names only, and the header names must be in lowercase.
// The order of the headers indicates the order the data was appended to create the
// signature.
func (h *HMAC) Headers() string {
	return strings.Join(h.headers, ";")
}

// Key returns the key ID used to create the HMAC signature.
func (h *HMAC) Key() string {
	return h.keyID
}

// Authorization returns the full Authorization header that can be added to the
// request headers of an outgoing webhook request.
func (h *HMAC) Authorization() (_ string, err error) {
	var sig string
	if sig, err = h.Signature(); err != nil {
		return "", err
	}

	return fmt.Sprintf("HMAC sig=%s, nonce=%s, headers=%s, kid=%s", sig, h.Nonce(), h.Headers(), h.Key()), nil
}

// Token is parsed from an Authorization or Server-Authorization header and is used
// to verify the signature of the request.
type Token struct {
	headers   []string
	data      []byte
	keyID     string
	signature []byte
}

// Parse the token from an Authorization or Server-Authorization header.
func Parse(token string) (tok *Token, err error) {
	if !strings.HasPrefix(token, "HMAC ") {
		return nil, ErrInvalidHMACToken
	}

	token = strings.TrimPrefix(token, "HMAC ")
	parts := strings.Split(token, ",")
	kv := make(map[string]string, len(parts))

	for _, part := range parts {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			return nil, ErrInvalidHMACToken
		}
		kv[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}

	tok = &Token{}
	if sig, ok := kv["sig"]; ok {
		if tok.signature, err = base64.RawURLEncoding.DecodeString(sig); err != nil {
			return nil, fmt.Errorf("could not decode signature: %w", err)
		}
	} else {
		return nil, fmt.Errorf("missing signature in HMAC token")
	}

	if nonce, ok := kv["nonce"]; ok {
		if tok.data, err = base64.RawURLEncoding.DecodeString(nonce); err != nil {
			return nil, fmt.Errorf("could not decode nonce: %w", err)
		}
	} else {
		return nil, fmt.Errorf("missing nonce in HMAC token")
	}

	if headers, ok := kv["headers"]; ok {
		for _, header := range strings.Split(headers, ";") {
			if header = strings.TrimSpace(header); header != "" {
				// A colon in the name would make the signed "name:value" line ambiguous.
				if strings.Contains(header, ":") {
					return nil, fmt.Errorf("could not decode headers in HMAC token")
				}
				tok.headers = append(tok.headers, header)
			}
		}

		if len(tok.headers) == 0 {
			return nil, fmt.Errorf("could not decode headers in HMAC token")
		}
	} else {
		return nil, fmt.Errorf("missing headers in HMAC token")
	}

	if kid, ok := kv["kid"]; ok {
		tok.keyID = kid
		if tok.keyID == "" {
			return nil, fmt.Errorf("could not decode key ID in HMAC token")
		}
	} else {
		return nil, fmt.Errorf("missing key ID in HMAC token")
	}

	return tok, nil
}

func (h *Token) KeyID() string {
	return h.keyID
}

func (h *Token) Headers() []string {
	return h.headers
}

// Collect the data to verify from the headers of the request or the reply. A canonical
// "name:value\n" line is appended for every header in the token, in the order they are
// listed in the token. Only the first value of a header is collected, and a header
// that is missing is collected with an empty value.
func (h *Token) Collect(headers http.Header) {
	for _, header := range h.headers {
		h.data = canonical(h.data, strings.ToLower(header), headers.Get(header))
	}
}

func (h *Token) Verify(key []byte) (bool, error) {
	mac := hmac.New(sha256.New, key)
	if _, err := mac.Write(h.data); err != nil {
		return false, err
	}

	if !hmac.Equal(mac.Sum(nil), h.signature) {
		return false, nil
	}

	return true, nil
}

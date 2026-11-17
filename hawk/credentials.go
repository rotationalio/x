package hawk

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
)

var (
	ErrInvalidKeyID  = errors.New("hawk: key ID does not match client ID")
	ErrInvalidSecret = errors.New("hawk: invalid secret")
)

// The Credentialing interface is used to supply the key ID and secret for the request.
type Credentialing interface {
	KeyID(request *http.Request) string
	Key(keyID string) ([]byte, error)
}

// Static credentials that that implement the Credentials interface. Note that the
// secret should be store in base64 encoded form and should be 32 bytes long.
type Credential struct {
	ClientID string `json:"client_id" yaml:"client_id" msgpack:"client_id" env:"HAWK_CLIENT_ID"`
	Secret   string `json:"client_secret" yaml:"client_secret" msgpack:"client_secret" env:"HAWK_CLIENT_SECRET"`
}

var _ Credentialing = (*Credential)(nil)

func (c *Credential) KeyID(request *http.Request) string {
	return c.ClientID
}

func (c *Credential) Key(keyID string) ([]byte, error) {
	if c.ClientID == "" {
		return nil, ErrMissingKeyID
	}

	if keyID != c.ClientID {
		return nil, ErrInvalidKeyID
	}

	// The secret is decoded on every call rather than cached so that the credential is
	// safe for concurrent use and can be copied.
	return decodeSecret(c.Secret)
}

// An API Key is a string that separates the client ID and secret with a : and is used
// for one line/one string credentials to make it easier to pass to the application.
type APIKey string

var _ Credentialing = (*APIKey)(nil)

func (k APIKey) KeyID(request *http.Request) string {
	i := strings.Index(string(k), ":")
	if i == -1 {
		return ""
	}
	return string(k[:i])
}

func (k APIKey) Key(keyID string) ([]byte, error) {
	i := strings.Index(string(k), ":")
	if i == -1 {
		return nil, ErrInvalidSecret
	}

	if i == 0 {
		return nil, ErrMissingKeyID
	}

	if keyID != string(k[:i]) {
		return nil, ErrInvalidKeyID
	}
	return decodeSecret(string(k[i+1:]))
}

// The minimum length of a decoded secret in bytes (the size of an HMAC-SHA256 digest).
const minSecretLength = 32

// Decodes the base64 encoded secret and rejects secrets that are empty or too short;
// no partially decoded secret is returned with an error.
func decodeSecret(encoded string) ([]byte, error) {
	secret, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	if len(secret) < minSecretLength {
		return nil, ErrInvalidSecret
	}
	return secret, nil
}

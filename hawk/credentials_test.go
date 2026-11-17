package hawk_test

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/hawk"
)

// cSpell: disable
const (
	credKeyID = "01JT4B3R5Z6AHJXV87QHPPKRBM"

	// The same 32 byte key in hex and in unpadded, standard alphabet base64.
	credKeyHex = "cfbabc4715b4759d45ba26953dd2fc0bfc2344ef70a2005432e7f16b5081610d"
	credSecret = "z7q8RxW0dZ1FuiaVPdL8C/wjRO9wogBUMufxa1CBYQ0"
)

// Secrets that are not unpadded, standard alphabet base64 and must be rejected.
var badSecrets = map[string]string{
	"padded":             credSecret + "=",
	"url safe alphabet":  strings.ReplaceAll(credSecret, "/", "_"),
	"illegal characters": "not base64!",
	"invalid prefix":     "!" + credSecret,
	"invalid suffix":     credSecret + "!",
	"incomplete quantum": "z",
	"surrounding space":  " " + credSecret + " ",
}

// Key IDs that must not be accepted in place of credKeyID.
var wrongKeyIDs = map[string]string{
	"empty":          "",
	"other":          "01JT4B3R5Z6AHJXV87QHPPKRBN",
	"lowercase":      strings.ToLower(credKeyID),
	"prefix":         credKeyID[:len(credKeyID)-1],
	"longer":         credKeyID + "0",
	"whitespace":     " " + credKeyID,
	"trailing colon": credKeyID + ":",
	"api key":        credKeyID + ":" + credSecret,
}

// cSpell: enable

// Decoded secret lengths on either side of the 32 byte minimum.
var (
	shortSecretLengths = []int{0, 1, 16, 31}
	longSecretLengths  = []int{33, 64}
)

func credKey(t *testing.T) []byte {
	t.Helper()
	key, err := hex.DecodeString(credKeyHex)
	assert.Ok(t, err, "could not decode key fixture")
	return key
}

// Returns a key of n bytes (up to 64) and its unpadded, standard alphabet base64.
func credKeyOfLength(t *testing.T, n int) (key []byte, secret string) {
	t.Helper()
	key = bytes.Repeat(credKey(t), 2)[:n]
	return key, base64.RawStdEncoding.EncodeToString(key)
}

func TestCredentialing(t *testing.T) {
	// Every implementation must supply the same key ID and key for the request so
	// that the request is signed with a signature the server can verify.
	tests := map[string]hawk.Credentialing{
		"Credential": &hawk.Credential{ClientID: credKeyID, Secret: credSecret},
		"APIKey":     hawk.APIKey(credKeyID + ":" + credSecret),
	}

	for name, creds := range tests {
		t.Run(name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, "https://example.com/webhook", nil)
			assert.Ok(t, err, "could not create request")
			req.Header.Set("X-Transaction-ID", "f17c9693-c280-4836-b544-245d832a11e0")

			keyID := creds.KeyID(req)
			assert.Equal(t, credKeyID, keyID, "key ID does not match")

			key, err := creds.Key(keyID)
			assert.Ok(t, err, "could not get key")
			assert.Equal(t, credKey(t), key, "key does not match")

			mac := hawk.New(keyID, key)
			mac.Append("X-Transaction-ID", req.Header.Get("X-Transaction-ID"))

			auth, err := mac.Authorization()
			assert.Ok(t, err, "could not create authorization header")

			token, err := hawk.Parse(auth)
			assert.Ok(t, err, "could not parse authorization header")
			assert.Equal(t, credKeyID, token.KeyID(), "token key ID does not match")

			token.Collect(req.Header)
			ok, err := token.Verify(credKey(t))
			assert.Ok(t, err, "could not verify token")
			assert.True(t, ok, "token verification failed")
		})
	}
}

func TestCredential(t *testing.T) {
	t.Run("KeyID", func(t *testing.T) {
		creds := &hawk.Credential{ClientID: credKeyID, Secret: credSecret}
		assert.Equal(t, credKeyID, creds.KeyID(nil), "key ID should not depend on the request")

		creds = &hawk.Credential{}
		assert.Equal(t, "", creds.KeyID(nil), "expected no key ID without a client ID")
	})

	t.Run("Key", func(t *testing.T) {
		// The key must be the same on every call, not just the first one.
		creds := &hawk.Credential{ClientID: credKeyID, Secret: credSecret}
		for i := range 3 {
			key, err := creds.Key(credKeyID)
			assert.Ok(t, err, "call %d: could not get key", i)
			assert.Equal(t, credKey(t), key, "call %d: key does not match", i)
		}
	})

	t.Run("WrongKeyID", func(t *testing.T) {
		for name, keyID := range wrongKeyIDs {
			t.Run(name, func(t *testing.T) {
				// Check both before and after the correct key ID has been used.
				creds := &hawk.Credential{ClientID: credKeyID, Secret: credSecret}
				for i := range 2 {
					key, err := creds.Key(keyID)
					assert.ErrorIs(t, err, hawk.ErrInvalidKeyID, "call %d: unexpected error", i)
					assert.Nil(t, key, "call %d: expected no key", i)

					_, err = creds.Key(credKeyID)
					assert.Ok(t, err, "call %d: could not get key with correct key ID", i)
				}
			})
		}
	})

	t.Run("EmptyKeyID", func(t *testing.T) {
		// Without a client ID no key ID can be correct, including the empty one.
		tests := map[string]*hawk.Credential{
			"zero value":   {},
			"no client ID": {Secret: credSecret},
		}

		for name, creds := range tests {
			t.Run(name, func(t *testing.T) {
				for _, keyID := range []string{creds.KeyID(nil), credKeyID} {
					key, err := creds.Key(keyID)
					assert.ErrorIs(t, err, hawk.ErrMissingKeyID, "key ID %q: unexpected error", keyID)
					assert.Nil(t, key, "key ID %q: expected no key", keyID)
				}
			})
		}
	})

	t.Run("BadSecret", func(t *testing.T) {
		for name, secret := range badSecrets {
			t.Run(name, func(t *testing.T) {
				// A bad secret must be rejected on every call, not just the first one.
				creds := &hawk.Credential{ClientID: credKeyID, Secret: secret}
				for i := range 3 {
					key, err := creds.Key(credKeyID)
					assert.Error(t, err, "call %d: expected error decoding secret", i)
					assert.Nil(t, key, "call %d: expected no key", i)
				}
			})
		}
	})

	t.Run("ShortSecret", func(t *testing.T) {
		for _, n := range shortSecretLengths {
			t.Run(fmt.Sprintf("%d bytes", n), func(t *testing.T) {
				// A short secret must be rejected on every call, not just the first one.
				_, secret := credKeyOfLength(t, n)
				creds := &hawk.Credential{ClientID: credKeyID, Secret: secret}
				for i := range 3 {
					key, err := creds.Key(credKeyID)
					assert.ErrorIs(t, err, hawk.ErrInvalidSecret, "call %d: unexpected error", i)
					assert.Nil(t, key, "call %d: expected no key", i)
				}
			})
		}
	})

	t.Run("LongSecret", func(t *testing.T) {
		for _, n := range longSecretLengths {
			t.Run(fmt.Sprintf("%d bytes", n), func(t *testing.T) {
				expected, secret := credKeyOfLength(t, n)
				creds := &hawk.Credential{ClientID: credKeyID, Secret: secret}

				key, err := creds.Key(credKeyID)
				assert.Ok(t, err, "could not get key")
				assert.Equal(t, expected, key, "key does not match")
			})
		}
	})

	t.Run("SecretChanged", func(t *testing.T) {
		// The secret is decoded on every call, so a change to the secret must not be
		// ignored in favor of a secret that was decoded by an earlier call.
		creds := &hawk.Credential{ClientID: credKeyID, Secret: credSecret}
		key, err := creds.Key(credKeyID)
		assert.Ok(t, err, "could not get key")
		assert.Equal(t, credKey(t), key, "key does not match")

		expected, secret := credKeyOfLength(t, 64)
		creds.Secret = secret
		key, err = creds.Key(credKeyID)
		assert.Ok(t, err, "could not get key for changed secret")
		assert.Equal(t, expected, key, "key does not match changed secret")

		creds.Secret = "not base64!"
		key, err = creds.Key(credKeyID)
		assert.Error(t, err, "expected error decoding changed secret")
		assert.Nil(t, key, "expected no key")

		creds.Secret = credSecret
		key, err = creds.Key(credKeyID)
		assert.Ok(t, err, "could not get key for restored secret")
		assert.Equal(t, credKey(t), key, "key does not match restored secret")
	})

	t.Run("KeyNotShared", func(t *testing.T) {
		// Modifying a key that was returned must not modify the next key returned.
		creds := &hawk.Credential{ClientID: credKeyID, Secret: credSecret}
		key, err := creds.Key(credKeyID)
		assert.Ok(t, err, "could not get key")
		clear(key)

		key, err = creds.Key(credKeyID)
		assert.Ok(t, err, "could not get key")
		assert.Equal(t, credKey(t), key, "key was modified through an earlier key")
	})

	t.Run("Concurrent", func(t *testing.T) {
		// A credential must be safe for concurrent use by multiple goroutines since it
		// is shared by all of the requests of a transport.
		const workers = 16
		creds := &hawk.Credential{ClientID: credKeyID, Secret: credSecret}

		var wg sync.WaitGroup
		keys := make([][]byte, workers)
		errs := make([]error, workers)
		for i := range workers {
			wg.Go(func() {
				keys[i], errs[i] = creds.Key(creds.KeyID(nil))
			})
		}
		wg.Wait()

		for i := range workers {
			assert.Ok(t, errs[i], "worker %d: could not get key", i)
			assert.Equal(t, credKey(t), keys[i], "worker %d: key does not match", i)
		}
	})

	t.Run("JSON", func(t *testing.T) {
		data := `{"client_id": "` + credKeyID + `", "client_secret": "` + credSecret + `"}`

		creds := &hawk.Credential{}
		assert.Ok(t, json.Unmarshal([]byte(data), creds), "could not unmarshal credential")
		assert.Equal(t, credKeyID, creds.ClientID, "client ID does not match")
		assert.Equal(t, credSecret, creds.Secret, "secret does not match")

		key, err := creds.Key(creds.KeyID(nil))
		assert.Ok(t, err, "could not get key")
		assert.Equal(t, credKey(t), key, "key does not match")
	})
}

func TestAPIKey(t *testing.T) {
	t.Run("KeyID", func(t *testing.T) {
		tests := []struct {
			name   string
			apiKey hawk.APIKey
			keyID  string
		}{
			{"valid", hawk.APIKey(credKeyID + ":" + credSecret), credKeyID},
			{"split on first colon", hawk.APIKey(credKeyID + ":" + credSecret + ":extra"), credKeyID},
			{"bad secret", hawk.APIKey(credKeyID + ":not base64!"), credKeyID},
			{"no colon", hawk.APIKey(credKeyID + credSecret), ""},
			{"empty", hawk.APIKey(""), ""},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				assert.Equal(t, tc.keyID, tc.apiKey.KeyID(nil), "key ID does not match")
			})
		}
	})

	t.Run("Key", func(t *testing.T) {
		apiKey := hawk.APIKey(credKeyID + ":" + credSecret)
		for i := range 3 {
			key, err := apiKey.Key(credKeyID)
			assert.Ok(t, err, "call %d: could not get key", i)
			assert.Equal(t, credKey(t), key, "call %d: key does not match", i)
		}
	})

	t.Run("BadFormat", func(t *testing.T) {
		tests := map[string]hawk.APIKey{
			"empty":    hawk.APIKey(""),
			"no colon": hawk.APIKey(credKeyID + credSecret),
			"id only":  hawk.APIKey(credKeyID),
		}

		for name, apiKey := range tests {
			t.Run(name, func(t *testing.T) {
				key, err := apiKey.Key(apiKey.KeyID(nil))
				assert.ErrorIs(t, err, hawk.ErrInvalidSecret, "unexpected error")
				assert.Nil(t, key, "expected no key")
			})
		}
	})

	t.Run("WrongKeyID", func(t *testing.T) {
		apiKey := hawk.APIKey(credKeyID + ":" + credSecret)
		for name, keyID := range wrongKeyIDs {
			t.Run(name, func(t *testing.T) {
				key, err := apiKey.Key(keyID)
				assert.ErrorIs(t, err, hawk.ErrInvalidKeyID, "unexpected error")
				assert.Nil(t, key, "expected no key")
			})
		}
	})

	t.Run("EmptyKeyID", func(t *testing.T) {
		// Without a client ID no key ID can be correct, including the empty one.
		tests := map[string]hawk.APIKey{
			"colon only":   hawk.APIKey(":"),
			"no client ID": hawk.APIKey(":" + credSecret),
		}

		for name, apiKey := range tests {
			t.Run(name, func(t *testing.T) {
				for _, keyID := range []string{apiKey.KeyID(nil), credKeyID} {
					key, err := apiKey.Key(keyID)
					assert.ErrorIs(t, err, hawk.ErrMissingKeyID, "key ID %q: unexpected error", keyID)
					assert.Nil(t, key, "key ID %q: expected no key", keyID)
				}
			})
		}
	})

	t.Run("BadSecret", func(t *testing.T) {
		tests := map[string]string{"extra colon": credSecret + ":extra"}
		for name, secret := range badSecrets {
			tests[name] = secret
		}

		for name, secret := range tests {
			t.Run(name, func(t *testing.T) {
				apiKey := hawk.APIKey(credKeyID + ":" + secret)
				key, err := apiKey.Key(credKeyID)
				assert.Error(t, err, "expected error decoding secret")
				assert.Nil(t, key, "expected no key")
			})
		}
	})

	t.Run("ShortSecret", func(t *testing.T) {
		for _, n := range shortSecretLengths {
			t.Run(fmt.Sprintf("%d bytes", n), func(t *testing.T) {
				_, secret := credKeyOfLength(t, n)
				apiKey := hawk.APIKey(credKeyID + ":" + secret)

				key, err := apiKey.Key(credKeyID)
				assert.ErrorIs(t, err, hawk.ErrInvalidSecret, "unexpected error")
				assert.Nil(t, key, "expected no key")
			})
		}
	})

	t.Run("LongSecret", func(t *testing.T) {
		for _, n := range longSecretLengths {
			t.Run(fmt.Sprintf("%d bytes", n), func(t *testing.T) {
				expected, secret := credKeyOfLength(t, n)
				apiKey := hawk.APIKey(credKeyID + ":" + secret)

				key, err := apiKey.Key(credKeyID)
				assert.Ok(t, err, "could not get key")
				assert.Equal(t, expected, key, "key does not match")
			})
		}
	})
}

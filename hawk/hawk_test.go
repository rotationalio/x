package hawk_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/hawk"
)

func TestHMAC(t *testing.T) {
	// cSpell: disable
	key, _ := hex.DecodeString("cfbabc4715b4759d45ba26953dd2fc0bfc2344ef70a2005432e7f16b5081610d")
	keyID := "01JT4B3R5Z6AHJXV87QHPPKRBM"
	// cSpell: enable

	headers := make(http.Header)
	headers.Set("Host", "example.com")
	headers.Set("X-Transaction-ID", "f17c9693-c280-4836-b544-245d832a11e0")

	mac := hawk.New(keyID, key)
	mac.Append("Host", headers.Get("Host"))
	mac.Append("X-Transaction-ID", headers.Get("X-Transaction-ID"))

	sig, err := mac.Signature()
	assert.Ok(t, err, "could not create signature")

	assert.Regexp(t, regexp.MustCompile(`^[A-Za-z0-9\-_]{22}$`), mac.Nonce(), "nonce is not 16 base64 encoded bytes")
	assert.Regexp(t, regexp.MustCompile(`^[A-Za-z0-9\-_]{43}$`), sig, "signature is not 32 base64 encoded bytes")
	assert.Equal(t, "host;x-transaction-id", mac.Headers(), "headers do not match")
	assert.Equal(t, keyID, mac.Key(), "key ID does not match")

	auth, err := mac.Authorization()
	assert.Ok(t, err, "could not create authorization header")
	assert.True(t, strings.HasPrefix(auth, "HMAC "))

	token, err := hawk.Parse(auth)
	assert.Ok(t, err, "could not parse authorization header")
	assert.Equal(t, mac.Key(), token.KeyID(), "key ID does not match")

	token.Collect(headers)
	ok, err := token.Verify(key)
	assert.Ok(t, err, "could not verify token")
	assert.True(t, ok, "token verification failed")
}

func TestInvalidHMAC(t *testing.T) {
	// cSpell: disable
	key, _ := hex.DecodeString("cfbabc4715b4759d45ba26953dd2fc0bfc2344ef70a2005432e7f16b5081610d")
	keyID := "01JT4B3R5Z6AHJXV87QHPPKRBM"
	// cSpell: enable

	headers := make(http.Header)
	headers.Set("Host", "example.com")
	headers.Set("X-Transaction-ID", "f17c9693-c280-4836-b544-245d832a11e0")

	mac := hawk.New(keyID, key)
	for header := range headers {
		mac.Append(header, headers.Get(header))
	}

	auth, err := mac.Authorization()
	assert.Ok(t, err, "could not create authorization header")

	token, err := hawk.Parse(auth)
	assert.Ok(t, err, "could not parse authorization header")

	token.Collect(headers)

	ok, err := token.Verify([]byte("bad key"))
	assert.Ok(t, err, "could not verify token with bad key")
	assert.False(t, ok, "token verification should fail with bad key")
}

func TestBadTokens(t *testing.T) {
	// Test invalid tokens
	// cSpell: disable
	tokens := []struct {
		token string
		err   error
	}{
		{
			"foo", hawk.ErrInvalidHMACToken,
		},
		{
			"", hawk.ErrInvalidHMACToken,
		},
		{
			"HMAC", hawk.ErrInvalidHMACToken,
		},
		{
			"HMAC ", hawk.ErrInvalidHMACToken,
		},
		{
			"HMAC foo", hawk.ErrInvalidHMACToken,
		},
		{
			"HMAC sig=zFeIxdyVnJtpMUExK7HoL37VN4tF6sMQZPEr58MBpMQ, nonce, headers=host;x-transaction-id, kid=01JT4B3R5Z6AHJXV87QHPPKRBM",
			hawk.ErrInvalidHMACToken,
		},
		{
			"HMAC sig=zFeIxdyVnJtpMUExK7HoL37VN4tF6sMQZPEr58MBpMQ, nonce=wm0eHegl4lD0Uw_sSYYQCw, headers=host;x-transaction-id, kid=01JT4B3R5Z6AHJXV87QHPPKRBM,",
			hawk.ErrInvalidHMACToken,
		},
		{
			"HMAC sig=zFeIxdyVnJtpMUExK7HoL37VN4tF6sMQZPEr58MBpMQ, nonce=wm0eHegl4lD0Uw_sSYYQCw, headers=; ;;, kid=01JT4B3R5Z6AHJXV87QHPPKRBM",
			hawk.ErrDecodeHeaders,
		},
		{
			"HMAC sig=zFeIxdyVnJtpMUExK7HoL37VN4tF6sMQZPEr58MBpMQ, nonce=wm0eHegl4lD0Uw_sSYYQCw, headers=host;x-transaction-id:foo, kid=01JT4B3R5Z6AHJXV87QHPPKRBM",
			hawk.ErrDecodeHeaders,
		},
		{
			"HMAC sig=f+, nonce=wm0eHegl4lD0Uw_sSYYQCw, headers=host;x-transaction-id, kid=01JT4B3R5Z6AHJXV87QHPPKRBM",
			hawk.ErrDecodeSignature,
		},
		{
			"HMAC nonce=wm0eHegl4lD0Uw_sSYYQCw, headers=host;x-transaction-id, kid=01JT4B3R5Z6AHJXV87QHPPKRBM",
			hawk.ErrMissingSignature,
		},
		{
			"HMAC sig=zFeIxdyVnJtpMUExK7HoL37VN4tF6sMQZPEr58MBpMQ, nonce=f+, headers=host;x-transaction-id, kid=01JT4B3R5Z6AHJXV87QHPPKRBM",
			hawk.ErrDecodeNonce,
		},
		{
			"HMAC sig=zFeIxdyVnJtpMUExK7HoL37VN4tF6sMQZPEr58MBpMQ, headers=host;x-transaction-id, kid=01JT4B3R5Z6AHJXV87QHPPKRBM",
			hawk.ErrMissingNonce,
		},
		{
			"HMAC sig=zFeIxdyVnJtpMUExK7HoL37VN4tF6sMQZPEr58MBpMQ, nonce=wm0eHegl4lD0Uw_sSYYQCw, headers=, kid=01JT4B3R5Z6AHJXV87QHPPKRBM",
			hawk.ErrDecodeHeaders,
		},
		{
			"HMAC sig=zFeIxdyVnJtpMUExK7HoL37VN4tF6sMQZPEr58MBpMQ, nonce=wm0eHegl4lD0Uw_sSYYQCw, kid=01JT4B3R5Z6AHJXV87QHPPKRBM",
			hawk.ErrMissingHeaders,
		},
		{
			"HMAC sig=zFeIxdyVnJtpMUExK7HoL37VN4tF6sMQZPEr58MBpMQ, nonce=wm0eHegl4lD0Uw_sSYYQCw, headers=host;x-transaction-id, kid=",
			hawk.ErrDecodeKeyID,
		},
		{
			"HMAC sig=zFeIxdyVnJtpMUExK7HoL37VN4tF6sMQZPEr58MBpMQ, nonce=wm0eHegl4lD0Uw_sSYYQCw, headers=host;x-transaction-id",
			hawk.ErrMissingKeyID,
		},
	}
	// cSpell: enable

	for i, tc := range tokens {
		tok, err := hawk.Parse(tc.token)
		assert.ErrorIs(t, err, tc.err, "test case %d: expected error %q, got %q", i, tc.err, err)
		assert.Nil(t, tok, "test case %d: expected nil token, got %v", i, tok)
	}
}

func TestExtraKeys(t *testing.T) {
	// cSpell: disable
	token := "HMAC foo=bar, sig=zFeIxdyVnJtpMUExK7HoL37VN4tF6sMQZPEr58MBpMQ, nonce=wm0eHegl4lD0Uw_sSYYQCw, headers=host;x-transaction-id, kid=01JT4B3R5Z6AHJXV87QHPPKRBM,color=red"
	// cSpell: enable

	_, err := hawk.Parse(token)
	assert.Ok(t, err, "could not parse token with extra keys")
}

func TestEmptyHeaders(t *testing.T) {
	// Empty entries in the headers list should be ignored without panicking.
	expected := []string{"host", "x-transaction-id"}
	tests := []string{
		"host;;x-transaction-id",
		"host;;;x-transaction-id",
		";host;x-transaction-id",
		"host;x-transaction-id;",
		";;host; ;x-transaction-id;;",
		" host ; x-transaction-id ",
	}

	for i, headers := range tests {
		// cSpell: disable
		token := "HMAC sig=zFeIxdyVnJtpMUExK7HoL37VN4tF6sMQZPEr58MBpMQ, nonce=wm0eHegl4lD0Uw_sSYYQCw, headers=" + headers + ", kid=01JT4B3R5Z6AHJXV87QHPPKRBM"
		// cSpell: enable

		tok, err := hawk.Parse(token)
		assert.Ok(t, err, "test case %d: could not parse token", i)
		assert.Equal(t, expected, tok.Headers(), "test case %d: headers do not match", i)
	}
}

func TestCanonicalData(t *testing.T) {
	// The signature must be computed over the nonce followed by one "name:value\n"
	// line for each header, in the order that the headers were appended.
	// cSpell: disable
	key, _ := hex.DecodeString("cfbabc4715b4759d45ba26953dd2fc0bfc2344ef70a2005432e7f16b5081610d")
	keyID := "01JT4B3R5Z6AHJXV87QHPPKRBM"
	// cSpell: enable

	mac := hawk.New(keyID, key)
	mac.Append("Host", "example.com")
	mac.Append("X-Transaction-ID", "f17c9693-c280-4836-b544-245d832a11e0")

	nonce, err := base64.RawURLEncoding.DecodeString(mac.Nonce())
	assert.Ok(t, err, "could not decode nonce")

	expected := hmac.New(sha256.New, key)
	expected.Write(nonce)
	expected.Write([]byte("host:example.com\nx-transaction-id:f17c9693-c280-4836-b544-245d832a11e0\n"))

	sig, err := mac.Signature()
	assert.Ok(t, err, "could not create signature")
	assert.Equal(t, base64.RawURLEncoding.EncodeToString(expected.Sum(nil)), sig, "signature is not computed over the canonical data")
}

func TestCollect(t *testing.T) {
	// cSpell: disable
	key, _ := hex.DecodeString("cfbabc4715b4759d45ba26953dd2fc0bfc2344ef70a2005432e7f16b5081610d")
	keyID := "01JT4B3R5Z6AHJXV87QHPPKRBM"
	// cSpell: enable

	// Every test case is signed with X-Alpha: ab and X-Bravo: c in that order.
	tests := []struct {
		name     string
		list     string      // replaces the headers list in the token if not empty
		received http.Header // the headers that the token is verified against
		valid    bool
	}{
		{
			"unmodified", "",
			http.Header{"X-Alpha": {"ab"}, "X-Bravo": {"c"}}, true,
		},
		{
			"mixed case headers list", "X-Alpha;X-BRAVO",
			http.Header{"X-Alpha": {"ab"}, "X-Bravo": {"c"}}, true,
		},
		{
			"only the first value is collected", "",
			http.Header{"X-Alpha": {"ab", "zz"}, "X-Bravo": {"c"}}, true,
		},
		{
			"first value is not the signed value", "",
			http.Header{"X-Alpha": {"zz", "ab"}, "X-Bravo": {"c"}}, false,
		},
		{
			"characters shifted between values", "",
			http.Header{"X-Alpha": {"a"}, "X-Bravo": {"bc"}}, false,
		},
		{
			"values swapped and headers list reordered", "x-bravo;x-alpha",
			http.Header{"X-Alpha": {"c"}, "X-Bravo": {"ab"}}, false,
		},
		{
			"headers list reordered", "x-bravo;x-alpha",
			http.Header{"X-Alpha": {"ab"}, "X-Bravo": {"c"}}, false,
		},
		{
			"header renamed", "x-alpha;x-charlie",
			http.Header{"X-Alpha": {"ab"}, "X-Charlie": {"c"}}, false,
		},
		{
			"values merged into one header", "x-alpha",
			http.Header{"X-Alpha": {"abc"}}, false,
		},
		{
			"signed header missing", "",
			http.Header{"X-Alpha": {"abc"}}, false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mac := hawk.New(keyID, key)
			mac.Append("X-Alpha", "ab")
			mac.Append("X-Bravo", "c")

			auth, err := mac.Authorization()
			assert.Ok(t, err, "could not create authorization header")

			if tc.list != "" {
				auth = strings.Replace(auth, "headers="+mac.Headers(), "headers="+tc.list, 1)
			}

			token, err := hawk.Parse(auth)
			assert.Ok(t, err, "could not parse authorization header")

			token.Collect(tc.received)
			ok, err := token.Verify(key)
			assert.Ok(t, err, "could not verify token")
			assert.Equal(t, tc.valid, ok, "unexpected verification result")
		})
	}
}

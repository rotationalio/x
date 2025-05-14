package vero_test

import (
	"bytes"
	"crypto/rand"
	"testing"
	"time"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/vero"
)

func TestVerification(t *testing.T) {
	// Generate verification token and signature
	token, err := vero.NewToken(makeRecordId(), time.Now().Add(1*time.Hour))
	assert.Nil(t, err, "could not create token")
	verify, signature, err := token.Sign()
	assert.Nil(t, err, "could not sign token")

	// Create tokens string to send to user; assume that the signature is saved to db and loaded again
	tks := verify.String()

	// Pretend to save the signature to the database by marshaling it
	dbd, err := signature.MarshalBinary()
	assert.Nil(t, err, "could not marshal signature")

	// Parse the incoming token from the user
	parsed_token, err := vero.ParseVerification(tks)
	assert.Nil(t, err, "could not parse verification token")

	// Pretend to load the signature from the database by unmarshaling it
	signature = &vero.SignedToken{}
	err = signature.UnmarshalBinary(dbd)
	assert.Nil(t, err, "could not unmarshal signature")

	// Complete the workflow by verifying that everything is correct and secure
	secure, err := signature.Verify(parsed_token)
	assert.Nil(t, err, "could not verify token")
	assert.True(t, secure, "verification returned false")
}

func TestNewToken(t *testing.T) {
	t.Run("RequiresExpiration", func(t *testing.T) {
		token, err := vero.NewToken(makeRecordId(), time.Time{})
		assert.ErrorIs(t, err, vero.ErrInvalidExpiration, "no error for zero value expiration")
		assert.Nil(t, token, "token should be nil")
	})

	t.Run("NonceGeneration", func(t *testing.T) {
		token, err := vero.NewToken(makeRecordId(), time.Now().Add(1*time.Minute))
		assert.Nil(t, err, "could not create token")
		data, err := token.MarshalBinary()
		assert.Nil(t, err, "could not marshal binary")

		// Expects nonce length to be 64!
		assert.NotEqual(t, bytes.Repeat([]byte{0x0}, 64), data[len(data)-64:], "zero-valued nonce!")
	})

	t.Run("Randomness", func(t *testing.T) {
		recordID := makeRecordId()
		expiration := time.Now().Add(1 * time.Hour)

		// Generate 16 tokens with the same recordID and expiration timestamp
		tokens := make([]*vero.Token, 0, 16)
		for i := 0; i < 16; i++ {
			token, err := vero.NewToken(recordID, expiration)
			assert.Nil(t, err, "could not create token")
			tokens = append(tokens, token)
		}

		// Ensure that all marshaled tokens are different (because of the nonce)
		for i, alpha := range tokens {
			for j, bravo := range tokens {
				if i == j {
					// Don't compare the same token to itself
					continue
				}

				da, err := alpha.MarshalBinary()
				assert.Nil(t, err, "could not marshal token %d", i)

				db, err := bravo.MarshalBinary()
				assert.Nil(t, err, "could not marshal token %d", j)

				assert.False(t, bytes.Equal(da, db), "tokens %d and %d were identical!", i, j)
			}
		}

	})
}

func TestTokenExpiration(t *testing.T) {
	t.Run("Happy", func(t *testing.T) {
		token, err := vero.NewToken(makeRecordId(), time.Now().Add(1*time.Minute))
		assert.Nil(t, err, "could not create token")
		assert.False(t, token.IsExpired(), "token should not be expired")
	})

	t.Run("ExpirationInPast", func(t *testing.T) {
		token, err := vero.NewToken(makeRecordId(), time.Now().Add(-1*time.Minute))
		assert.ErrorIs(t, err, vero.ErrInvalidExpiration, "expiration in past should return an error")
		assert.Nil(t, token, "token should be nil")
	})

	t.Run("NoExpiration", func(t *testing.T) {
		token, err := vero.NewToken(makeRecordId(), time.Time{})
		assert.ErrorIs(t, err, vero.ErrInvalidExpiration, "lack of expiration should return an error")
		assert.Nil(t, token, "token should be nil")
	})
}

func TestTokenSign(t *testing.T) {
	t.Run("Happy", func(t *testing.T) {
		token, err := vero.NewToken(makeRecordId(), time.Now().Add(1*time.Minute))
		assert.Nil(t, err, "could not create token")
		verification, signature, err := token.Sign()
		assert.Nil(t, err, "could not sign token")
		assert.Len(t, verification, 16+64, "unexpected length of verification token (16 bytes + 64 byte secret)")
		assert.Len(t, signature.Signature(), 32, "unexpected length of hmac signature (32 bytes for sha256)")
	})

	t.Run("WithoutNonce", func(t *testing.T) {
		token := &vero.Token{RecordID: makeRecordId(), Expiration: time.Now().Add(1 * time.Minute)}
		verification, signature, err := token.Sign()
		assert.Nil(t, err, "could not sign token")
		assert.Len(t, verification, 16+64, "unexpected length of verification token (16 bytes + 64 byte secret)")
		assert.Len(t, signature.Signature(), 32, "unexpected length of hmac signature (32 bytes for sha256)")
	})

	t.Run("VerificationToken", func(t *testing.T) {
		recordID := makeRecordId()
		token, err := vero.NewToken(recordID, time.Now().Add(1*time.Minute))
		assert.Nil(t, err, "could not create token")
		verify, _, err := token.Sign()
		assert.Nil(t, err, "could not sign token")
		assert.Equal(t, recordID, verify.RecordID(), "expected record ID to match")
		assert.Len(t, verify.Secret(), 64, "expected secret to be 64 bytes long")
	})

	t.Run("Sad", func(t *testing.T) {
		testCases := []*vero.Token{
			{},                         // empty
			{RecordID: makeRecordId()}, // no expiration
			{RecordID: make([]byte, 16), Expiration: time.Now().Add(1 * time.Minute)}, // zeroed recordID
		}

		for i, token := range testCases {
			verify, signature, err := token.Sign()
			assert.NotNil(t, err, "expected an error on test case %d", i)
			assert.Nil(t, verify, "expected nil verification on test case %d", i)
			assert.Nil(t, signature, "expected nil signed token on test case %d", i)
		}
	})

	t.Run("SecretRandomness", func(t *testing.T) {
		// Create a token with constant nonce, record id, and expiration
		token, err := vero.NewToken(makeRecordId(), time.Now().Add(1*time.Hour))
		assert.Nil(t, err, "could not create token")

		// Create 16 verification tokens from the same token

		tokens := make([]vero.VerificationToken, 0, 16)
		signatures := make([]*vero.SignedToken, 0, 16)
		for i := 0; i < 16; i++ {
			verify, signed, err := token.Sign()
			assert.Nil(t, err, "could not sign token")

			tokens = append(tokens, verify)
			signatures = append(signatures, signed)
		}

		for i, alpha := range tokens {
			for j, bravo := range tokens {
				if i == j {
					// Don't compare the same token
					continue
				}

				// No two verification tokens and signatures should be the same
				assert.False(t, bytes.Equal(alpha, bravo), "verification token %d is equal to token %d", i, j)
				assert.False(t, bytes.Equal(signatures[i].Signature(), signatures[j].Signature()), "signature %d is equal to signature %d", i, j)
			}
		}

	})

}

func TestTokenBinary(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		testCases := []struct {
			recordID   []byte
			expiration time.Time
		}{
			{makeRecordId(), time.Now().Add(1 * time.Minute)},
			{makeRecordId(), time.Now().Add(312391 * time.Hour)},
		}

		for i, tc := range testCases {
			token, err := vero.NewToken(tc.recordID, tc.expiration)
			assert.Nil(t, err, "could not create token %d", i)

			data, err := token.MarshalBinary()
			assert.NotNil(t, data, "test case %d returned nil data", i)
			assert.Nil(t, err, "test case %d errored on marshal", i)

			cmpt := &vero.Token{}
			err = cmpt.UnmarshalBinary(data)
			assert.Nil(t, err, "test case %d errored on unmarshal", i)

			assert.True(t, token.Equal(cmpt), "deserialization mismatch for test case %d", i)
		}
	})

	t.Run("BadMarshal", func(t *testing.T) {
		testCases := []struct {
			token *vero.Token
			err   error
		}{
			{
				&vero.Token{RecordID: make([]byte, 16), Expiration: time.Now().Add(1 * time.Minute)},
				vero.ErrInvalidTokenRecordID,
			},
			{
				&vero.Token{RecordID: makeRecordId(), Expiration: time.Time{}},
				vero.ErrInvalidTokenExpiration,
			},
		}

		for i, tc := range testCases {
			data, err := tc.token.MarshalBinary()
			assert.Nil(t, data, "test case %d returned non-nil data", i)
			assert.ErrorIs(t, err, tc.err, "test case %d returned the wrong error", i)
		}
	})

	t.Run("BadUnmarshal", func(t *testing.T) {
		testCases := []struct {
			data []byte
			err  error
		}{
			{
				nil,
				vero.ErrTokenSize,
			},
			{
				[]byte{},
				vero.ErrTokenSize,
			},
			{
				[]byte{0x1, 0x2, 0x3, 0x4, 0xf, 0xfe},
				vero.ErrTokenSize,
			},
			{
				bytes.Repeat([]byte{0x1, 0x2, 0x3, 0x4, 0xf, 0xfe}, 64),
				vero.ErrTokenSize,
			},
			{
				[]byte{
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
				},
				vero.ErrDecode,
			},
			{
				[]byte{
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
					0xff, 0x00,
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d,
				},
				vero.ErrInvalidTokenNonce,
			},
		}

		for i, tc := range testCases {
			token := &vero.Token{}
			err := token.UnmarshalBinary(tc.data)
			assert.ErrorIs(t, err, tc.err, "test case %d returned the wrong error", i)
		}
	})
}

func TestSignedTokenBinary(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		testCases := []struct {
			recordID   []byte
			expiration time.Time
		}{
			{
				makeRecordId(),
				time.Now().Add(1 * time.Minute),
			},
			{
				makeRecordId(),
				time.Now().Add(312391 * time.Hour),
			},
		}

		for i, tc := range testCases {
			token, err := vero.NewToken(tc.recordID, tc.expiration)
			assert.Nil(t, err, "could not create token %d", i)

			_, signed, err := token.Sign()
			assert.Nil(t, err, "could not sign token %d", i)

			data, err := signed.MarshalBinary()
			assert.NotNil(t, data, "test case %d returned nil data", i)
			assert.Nil(t, err, "test case %d errored on marshal", i)

			cmpt := &vero.SignedToken{}
			err = cmpt.UnmarshalBinary(data)
			assert.Nil(t, err, "test case %d errored on unmarshal", i)

			assert.True(t, signed.Equal(cmpt), "deserialization mismatch for test case %d", i)
		}
	})

	t.Run("BadMarshal", func(t *testing.T) {
		testCases := []struct {
			token *vero.SignedToken
			err   error
		}{
			{
				&vero.SignedToken{},
				vero.ErrInvalidTokenSignature,
			},
		}

		for i, tc := range testCases {
			data, err := tc.token.MarshalBinary()
			assert.Nil(t, data, "test case %d returned non-nil data", i)
			assert.ErrorIs(t, err, tc.err, "test case %d returned the wrong error", i)
		}
	})

	t.Run("BadUnmarshal", func(t *testing.T) {
		testCases := []struct {
			data []byte
			err  error
		}{
			{
				nil,
				vero.ErrTokenSize,
			},
			{
				[]byte{},
				vero.ErrTokenSize,
			},
			{
				[]byte{0x1, 0x2, 0x3, 0x4, 0xf, 0xfe},
				vero.ErrTokenSize,
			},
			{
				bytes.Repeat([]byte{0x1, 0x2, 0x3, 0x4, 0xf, 0xfe}, 64),
				vero.ErrTokenSize,
			},
			{
				[]byte{
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
					0xff, 0x00,
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d,
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
				},
				vero.ErrInvalidTokenSignature,
			},
		}

		for i, tc := range testCases {
			token := &vero.SignedToken{}
			err := token.UnmarshalBinary(tc.data)
			assert.ErrorIs(t, err, tc.err, "test case %d returned the wrong error", i)
		}
	})
}

func TestVerificationToken(t *testing.T) {
	t.Run("Static", func(t *testing.T) {
		tks := "MTIzNDU2Nzg5MGFiY2RlZvRoC07KOs375xDclKlFe2gKk3TUcxj7-ID9TlccbGtE3dAEFjzOE9o2B9e-y_lNqkTVJfEPm3n8Kt-9gPQbU-E"
		token, err := vero.ParseVerification(tks)
		assert.Nil(t, err, "could not parse good verification token")
		assert.Equal(t, token.RecordID(), []byte("1234567890abcdef"), "unexpected record id")

		secret := []byte{
			0xF4, 0x68, 0x0B, 0x4E, 0xCA, 0x3A, 0xCD, 0xFB, 0xE7, 0x10, 0xDC, 0x94, 0xA9, 0x45, 0x7B, 0x68,
			0x0A, 0x93, 0x74, 0xD4, 0x73, 0x18, 0xFB, 0xF8, 0x80, 0xFD, 0x4E, 0x57, 0x1C, 0x6C, 0x6B, 0x44,
			0xDD, 0xD0, 0x04, 0x16, 0x3C, 0xCE, 0x13, 0xDA, 0x36, 0x07, 0xD7, 0xBE, 0xCB, 0xF9, 0x4D, 0xAA,
			0x44, 0xD5, 0x25, 0xF1, 0x0F, 0x9B, 0x79, 0xFC, 0x2A, 0xDF, 0xBD, 0x80, 0xF4, 0x1B, 0x53, 0xE1,
		}

		assert.Equal(t, secret, token.Secret(), "unexpected secret")
	})

	t.Run("TooShort", func(t *testing.T) {
		tks := "k0ZmbMJcQeyFtAtZ0_2EXMHwJ1ufcB4831ozVeHzAcVpyKybKzelG0l9qbJ4K5IUjaGSx5EdJ_"
		token, err := vero.ParseVerification(tks)
		assert.ErrorIs(t, err, vero.ErrTokenSize, "expected size parsing error")
		assert.Nil(t, token, "expected nil token returned")
	})

	t.Run("BadDecode", func(t *testing.T) {
		// '}' at char 19
		tks := "k0ZmbMJcQeyFtAtZ0_2}XMHwJ1ufcB4831ozVeHzAcVpyKybKzelG0l9qbJ4K5IUjaGSx5EdJ_"
		token, err := vero.ParseVerification(tks)
		assert.Equal(t, err.Error(), "illegal base64 data at input byte 19", "expected base64 parsing error")
		assert.Nil(t, token, "expected nil token returned")
	})
}

// Helpers

func makeRecordId() (out []byte) {
	out = make([]byte, 16)
	if _, err := rand.Read(out); err != nil {
		panic(err)
	}
	return out
}

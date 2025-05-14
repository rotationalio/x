package vero

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

const (
	nonceLength        = 64
	keyLength          = 64
	hmacLength         = 32
	recordIdLength     = 16 // 16 bytes fits a ULID
	minTokenLength     = recordIdLength + nonceLength + 1
	maxTokenLength     = recordIdLength + nonceLength + binary.MaxVarintLen64
	minSignTokenLength = minTokenLength + hmacLength
	maxSignTokenLength = maxTokenLength + hmacLength
	verifyTokenLength  = recordIdLength + keyLength
)

// A Token is a data representation of the information needed to create a secure
// record verification token for use in cases such as sending Sunrise email or
// password reset links. Tokens can be used to generate SignedTokens and
// SignedTokens can be used to send a secure verification token and to verify
// that tokens belong to the specified user.
type Token struct {
	RecordID   []byte    // ID of the record in the database for verification
	Expiration time.Time // Expiration date of the token (not after)
	nonce      []byte    // Random nonce for cryptographic security
}

// A signed token contains a signature that can be stored in the local database in
// order to verify an incoming verification token from a client.
type SignedToken struct {
	Token
	signature []byte // The HMAC signature computed from the Token data (read-only)
}

// A verification token is sent to the client and contains the information needed to
// lookup a signed token in the database and to verify that the message is authentic.
type VerificationToken []byte

//===========================================================================
// Token Methods
//===========================================================================

// Create a new token with the specified record ID and expiration timestamp. The
// `recordId` must be 16 bytes in length and the `expiration` must be in the future.
func NewToken(recordID []byte, expiration time.Time) (token *Token, err error) {
	if expiration.IsZero() || expiration.Before(time.Now()) {
		return nil, ErrInvalidExpiration
	}

	if recordIdLength != len(recordID) {
		return nil, ErrRecordIDSize
	}

	token = &Token{
		RecordID:   recordID,
		Expiration: expiration,
		nonce:      make([]byte, nonceLength),
	}

	if _, err := rand.Read(token.nonce); err != nil {
		panic(fmt.Errorf("no crypto random generator available: %w", err))
	}

	return token, nil
}

// Sign a token creating a verification token that should be sent as a string to the
// counterparty and a signed token that should be stored in the database.
func (t *Token) Sign() (token VerificationToken, signature *SignedToken, err error) {
	// Generate nonce if the token was instantiated without New
	if t.nonce == nil {
		t.nonce = make([]byte, nonceLength)
		if _, err := rand.Read(t.nonce); err != nil {
			panic(fmt.Errorf("no crypto random generator available: %w", err))
		}
	}

	// Create a random secret key for signing
	secret := make([]byte, keyLength)
	if _, err := rand.Read(secret); err != nil {
		panic(fmt.Errorf("no crypto random generator available: %w", err))
	}

	// Marshal the token for signing
	var data []byte
	if data, err = t.MarshalBinary(); err != nil {
		return nil, nil, err
	}

	// Create HMAC signature for the token
	mac := hmac.New(sha256.New, secret)
	if _, err = mac.Write(data); err != nil {
		return nil, nil, err
	}

	// Get the HMAC signature and append it to the verification data
	// NOTE: this must happen after HMAC signing!
	signature = &SignedToken{
		Token:     *t,
		signature: mac.Sum(nil),
	}

	// Create the verification token
	token = make(VerificationToken, verifyTokenLength)
	copy(token[0:16], t.RecordID[:])
	copy(token[16:], secret)

	return token, signature, nil
}

func (t *Token) IsExpired() bool {
	if t.Expiration.IsZero() {
		return true
	}
	return t.Expiration.Before(time.Now())
}

func (t *Token) MarshalBinary() ([]byte, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}

	data := make([]byte, maxTokenLength)
	copy(data[:16], t.RecordID[:])

	i := binary.PutVarint(data[16:], t.Expiration.UnixNano())
	l := 16 + i
	copy(data[l:], t.nonce)

	l = l + nonceLength
	return data[:l], nil
}

func (t *Token) UnmarshalBinary(data []byte) error {
	if _, err := t.readFrom(data); err != nil {
		return err
	}
	return t.Validate()
}

func (t *Token) readFrom(data []byte) (int, error) {
	if len(data) > maxTokenLength || len(data) < minTokenLength {
		return 0, ErrTokenSize
	}

	// Parse record ID
	t.RecordID = []byte(data[:16])

	// Parse expiration time
	exp, i := binary.Varint(data[16 : 16+binary.MaxVarintLen64])
	if i <= 0 {
		return 16, ErrDecode
	}
	t.Expiration = time.Unix(0, exp)

	// Read the nonce data
	l := 16 + i
	if len(data[l:]) < nonceLength {
		return l, ErrInvalidTokenNonce
	}
	t.nonce = data[l : l+nonceLength]

	// Validate the binary data
	return l + nonceLength, nil
}

func (t *Token) Validate() (err error) {
	if len(t.RecordID) != recordIdLength {
		err = errors.Join(err, ErrRecordIDSize)
	}

	empty := make([]byte, recordIdLength)
	if bytes.Equal(t.RecordID, empty) {
		err = errors.Join(err, ErrInvalidTokenRecordID)
	}

	if t.Expiration.IsZero() {
		err = errors.Join(err, ErrInvalidTokenExpiration)
	}

	if len(t.nonce) != nonceLength {
		err = errors.Join(err, ErrInvalidTokenNonce)
	}

	return err
}

func (t *Token) Equal(o *Token) bool {
	return bytes.Equal(t.RecordID[:], o.RecordID[:]) &&
		t.Expiration.Equal(o.Expiration) &&
		bytes.Equal(t.nonce, o.nonce)
}

//===========================================================================
// SignedToken Methods
//===========================================================================

// Verify that a signed token belongs with the associated verification token.
func (t *SignedToken) Verify(token VerificationToken) (secure bool, err error) {
	if len(token) != verifyTokenLength {
		return false, ErrTokenSize
	}

	// Compute the hash of the current token for verification
	var data []byte
	if data, err = t.Token.MarshalBinary(); err != nil {
		return false, err
	}

	// Generate the HMAC signature of the current token
	mac := hmac.New(sha256.New, token.Secret())
	if _, err := mac.Write(data); err != nil {
		return false, err
	}

	return bytes.Equal(t.signature, mac.Sum(nil)), nil
}

// Retrieve the signature from the signed token.
func (t *SignedToken) Signature() []byte {
	return t.signature
}

// Scan the signed token from a database query.
func (t *SignedToken) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	data, ok := value.([]byte)
	if !ok {
		return ErrUnexpectedType
	}

	return t.UnmarshalBinary(data)
}

// Produce a database value from the signed token for inserts/updates to database.
func (t *SignedToken) Value() (_ driver.Value, err error) {
	if t == nil {
		return nil, nil
	}

	var data []byte
	if data, err = t.MarshalBinary(); err != nil {
		return nil, err
	}
	return data, nil
}

func (t *SignedToken) MarshalBinary() (out []byte, err error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}

	var token []byte
	if token, err = t.Token.MarshalBinary(); err != nil {
		return nil, err
	}

	out = make([]byte, maxSignTokenLength)
	copy(out[:len(token)], token)
	copy(out[len(token):], t.signature)

	return out[:len(token)+len(t.signature)], nil
}

func (t *SignedToken) UnmarshalBinary(data []byte) (err error) {
	if len(data) > maxSignTokenLength || len(data) < minSignTokenLength {
		return ErrTokenSize
	}

	// Parse Token
	var n int
	if n, err = t.Token.readFrom(data[:maxTokenLength]); err != nil {
		return err
	}

	// Extract the signature as the unread part of the data
	t.signature = data[n:]

	// Validate the binary data
	return t.Validate()
}

func (t *SignedToken) Validate() (err error) {
	err = t.Token.Validate()

	if len(t.signature) != hmacLength {
		err = errors.Join(err, ErrInvalidTokenSignature)
	}

	return err
}

func (t *SignedToken) Equal(o *SignedToken) bool {
	return t.Token.Equal(&o.Token) && bytes.Equal(t.signature, o.signature)
}

//===========================================================================
// VerificationToken Methods
//===========================================================================

func ParseVerification(tks string) (_ VerificationToken, err error) {
	var token []byte
	if token, err = base64.RawURLEncoding.DecodeString(tks); err != nil {
		return nil, err
	}

	if len(token) != verifyTokenLength {
		return nil, ErrTokenSize
	}

	return VerificationToken(token), nil
}

func (v VerificationToken) RecordID() []byte {
	return []byte(v[:16])
}

func (v VerificationToken) Secret() []byte {
	return v[16:]
}

func (v VerificationToken) String() string {
	return base64.RawURLEncoding.EncodeToString(v)
}

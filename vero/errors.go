package vero

import "errors"

var (
	ErrDecode                 = errors.New("vero: could not decode token")
	ErrTokenSize              = errors.New("vero: invalid size for token")
	ErrRecordIDSize           = errors.New("vero: invalid size for record id")
	ErrInvalidExpiration      = errors.New("vero: must provide an expiration time in the future")
	ErrInvalidTokenRecordID   = errors.New("vero: invalid verification token: no record id")
	ErrInvalidTokenExpiration = errors.New("vero: invalid verification token: no expiration timestamp")
	ErrInvalidTokenNonce      = errors.New("vero: invalid verification token: incorrect nonce")
	ErrInvalidTokenSignature  = errors.New("vero: invalid verification token: incorrect hmac signature")
	ErrUnexpectedType         = errors.New("vero: could not scan non-bytes type")
)

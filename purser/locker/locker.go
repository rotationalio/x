/*
Package locker defines the crypto interfaces implemented by locker/vN editions.

Single-import clients use [purser.Locker], [purser.Sealer], [purser.Labeler], and [purser.Keyer]
(see purser aliases.go).
*/
package locker

// Sealer performs namespace-bound encrypt and decrypt on wire blobs.
type Sealer interface {
	// Seal encrypts the provided plaintext within the given namespace,
	// returning a ciphertext blob and an error, if any occurs.
	Seal(namespace string, plaintext []byte) (ciphertext []byte, err error)

	// Open decrypts the provided ciphertext within the given namespace,
	// returning the resulting plaintext and an error, if decoding fails.
	Open(namespace string, ciphertext []byte) (plaintext []byte, err error)
}

// Labeler exposes non-secret locker metadata used for routing.
type Labeler interface {
	// Edition returns the edition name of the locker (e.g., "v1").
	Edition() string

	// Version returns the version number associated with the locker.
	Version() uint8

	// Recipe returns the non-secret recipe information used in processing.
	Recipe() string

	// Context returns the contextual namespace for this locker.
	Context() string
}

// Keyer identifies a locker instance and parses key ids from ciphertext wire.
type Keyer interface {
	// KeyID returns the key identifier for this locker instance.
	KeyID() []byte

	// ParseKeyID parses the key identifier from the given ciphertext wire bytes.
	ParseKeyID(ciphertext []byte) (keyID []byte, err error)
}

// Locker is a complete edition implementation (v1, test lockers, etc.).
type Locker interface {
	Sealer
	Labeler
	Keyer
}

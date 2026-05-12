/*
Package identifier defines the [Identifier] interface for minting and validating row ids, plus reusable
implementations. The bundled [HexIdentifier] uses 16 random bytes encoded as 32 hex digits.
*/
package identifier

// Identifier mints canonical string ids, validates caller-supplied ids, and maps between string ids
// and raw storage keys. Implementations are passed into the root vault constructor together with storage.
type Identifier interface {
	// New returns a fresh id when storing a new secret.
	New() (id string, err error)

	// Parse rejects malformed ids before any storage read or write.
	Parse(id string) error

	// MarshalBinary encodes the canonical string id to opaque storage key bytes.
	MarshalBinary(id string) (src []byte, err error)

	// UnmarshalBinary decodes storage key bytes back to the canonical string id.
	UnmarshalBinary(src []byte) (id string, err error)
}

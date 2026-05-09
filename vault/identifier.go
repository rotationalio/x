package vault

// Identifier mints canonical string ids and validates caller-supplied ids.
type Identifier interface {
	// New generates a new canonical identifier string.
	New() (id string, err error)

	// Parse validates the provided id string; returns error if invalid.
	Parse(id string) error

	// IDFromBytes produces a canonical string id from the given raw bytes; error if invalid.
	IDFromBytes(src []byte) (id string, err error)

	// BytesFromID returns the raw byte representation of a canonical id string; error if invalid.
	BytesFromID(id string) (src []byte, err error)
}

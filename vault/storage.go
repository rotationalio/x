package vault

import "context"

// Storage abstracts persistence of opaque ciphertext keyed by (namespace, id)
// for [Vault]; implementations must enforce compare-and-swap (CAS) semantics
// and uniqueness.
type Storage interface {
	// Create inserts a new row; duplicate (namespace, id) must return an error
	// ([ErrDuplicateKey] if appropriate).
	Create(ctx context.Context, namespace, id string, ciphertext []byte) error

	// Get returns ciphertext for an existing row or [ErrNotFound].
	Get(ctx context.Context, namespace, id string) (ciphertext []byte, err error)

	// Replace overwrites ciphertext for an existing row; missing row returns
	// [ErrNotFound].
	Replace(ctx context.Context, namespace, id string, ciphertext []byte) error

	// Delete removes a row; missing keys should succeed (idempotent delete).
	Delete(ctx context.Context, namespace, id string) error

	// CompareAndSwap sets ciphertext to newCiphertext only if the stored value
	// equals oldCiphertext. Wrong oldCiphertext returns [ErrCASFailed]. Missing
	// row returns [ErrNotFound].
	CompareAndSwap(ctx context.Context, namespace, id string, oldCiphertext, newCiphertext []byte) error
}

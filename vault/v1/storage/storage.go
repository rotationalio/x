/*
Package storage defines the [Storage] interface for opaque sealed vault rows and reusable implementations
for tests and small programs (notably [MemStorage]).

[Storage] abstracts persistence keyed by (namespace, id). Ciphertext values are opaque blobs produced by the
vault; implementations should map driver-specific failures to the stable sentinels in package
go.rtnl.ai/x/vault/v1/errors (not found, duplicate key, CAS failed, storage) where practical.
*/
package storage

import "context"

// Storage persists ciphertext blobs; keys are opaque (namespace, id) pairs.
type Storage interface {
	// Create inserts a new row; duplicate (namespace, id) must fail (typically duplicate key sentinel).
	Create(ctx context.Context, namespace, id string, ciphertext []byte) error

	// Get returns the stored blob or a not-found sentinel.
	Get(ctx context.Context, namespace, id string) (ciphertext []byte, err error)

	// Replace overwrites ciphertext for an existing row; missing row returns not-found.
	Replace(ctx context.Context, namespace, id string, ciphertext []byte) error

	// Delete removes the row if present; missing row must return nil (idempotent).
	Delete(ctx context.Context, namespace, id string) error

	// CompareAndSwap sets newCiphertext only when the stored value equals oldCiphertext; wrong old
	// value returns CAS-failed and a missing row returns not-found.
	CompareAndSwap(ctx context.Context, namespace, id string, oldCiphertext, newCiphertext []byte) error
}

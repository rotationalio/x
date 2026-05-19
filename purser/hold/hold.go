/*
Package hold defines the Hold interface for opaque sealed rows and reusable implementations
for tests and small programs (notably MemHold).

Hold abstracts persistence keyed by (namespace, identifier). Ciphertext values are opaque blobs produced by lockers;
implementations should map driver-specific failures to the stable sentinels in package
go.rtnl.ai/x/purser/errors (not found, duplicate key, CAS failed, hold) where practical.
*/
package hold

import (
	"context"

	"go.rtnl.ai/x/purser/hold/identifier"
)

// Hold persists ciphertext blobs; keys are opaque (namespace, identifier) pairs.
type Hold interface {
	// Identifier returns the configured identifier strategy. Most callers do not need this accessor.
	Identifier() identifier.Identifier

	// Create inserts a new row with an internally generated identifier.
	Create(ctx context.Context, namespace string, ciphertext []byte) (identifier string, err error)

	// CreateWithIdentifier inserts a new row using a caller-provided identifier.
	CreateWithIdentifier(ctx context.Context, namespace, identifier string, ciphertext []byte) error

	// Get returns the stored blob or a not-found sentinel.
	Get(ctx context.Context, namespace, identifier string) (ciphertext []byte, err error)

	// Replace overwrites ciphertext for an existing row; missing row returns not-found.
	Replace(ctx context.Context, namespace, identifier string, ciphertext []byte) error

	// Delete removes the row if present; missing row must return nil (idempotent).
	Delete(ctx context.Context, namespace, identifier string) error

	// CompareAndSwap sets newCiphertext only when the stored value equals oldCiphertext; wrong old
	// value returns CAS-failed and a missing row returns not-found.
	CompareAndSwap(ctx context.Context, namespace, identifier string, oldCiphertext, newCiphertext []byte) error
}

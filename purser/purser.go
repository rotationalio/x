/*
Package purser defines the version-neutral Purser, Locker, and Keyring interfaces for opaque secret rows.
Concrete locker implementations live in versioned subpackages (for example go.rtnl.ai/x/purser/locker/v1).
*/
package purser

import "context"

//=============================================================================
// Purser
//=============================================================================

// Purser is the contract for row operations (seal, open, compare-and-swap, move, delete).
// Versioned locker packages are wired in by callers through a Keyring.
type Purser interface {
	// Store seals plaintext for namespace and returns a new opaque identifier.
	Store(ctx context.Context, namespace string, plaintext []byte) (identifier string, err error)

	// Retrieve loads and opens the row for (namespace, identifier).
	Retrieve(ctx context.Context, namespace, identifier string) (plaintext []byte, err error)

	// Update re-seals plaintext using the active locker and replaces row ciphertext.
	Update(ctx context.Context, namespace, identifier string, plaintext []byte) error

	// CompareAndSwap updates plaintext only when decrypted content equals currentPlain.
	CompareAndSwap(ctx context.Context, namespace, identifier string, currentPlain, newPlain []byte) error

	// MoveNamespace re-seals plaintext for newNamespace and deletes the old row.
	MoveNamespace(ctx context.Context, oldNamespace, newNamespace, identifier string) error

	// Delete removes a row for (namespace, identifier).
	Delete(ctx context.Context, namespace, identifier string) error
}

// Locker seals and opens opaque ciphertext for one long-term keypair.
type Locker interface {
	// KeyID returns the key identifier embedded into row metadata.
	KeyID() []byte

	// Seal encrypts plaintext into wire ciphertext bound to namespace.
	Seal(namespace string, plaintext []byte) (ciphertext []byte, err error)

	// Open decrypts ciphertext and verifies namespace binding.
	Open(namespace string, ciphertext []byte) (plaintext []byte, err error)

	// ParseKeyID reads the key identifier from ciphertext metadata without full decrypt.
	ParseKeyID(ciphertext []byte) (keyID []byte, err error)
}

// Keyring tracks the active locker for writes and all registered lockers for decrypt routing.
type Keyring interface {
	// Active returns the locker used for all write operations.
	Active() Locker

	// Lookup returns a locker by key identifier for decrypt routing.
	Lookup(keyID []byte) (Locker, bool)

	// Register adds a locker to the decrypt routing set.
	Register(Locker) error

	// SetActive switches the write locker (and registers it if needed).
	SetActive(Locker) error

	// RouteKeyID extracts a key identifier from ciphertext and returns the matching locker.
	// Implementations try each registered locker's ParseKeyID until one succeeds and the
	// extracted key ID maps to a registered locker.
	RouteKeyID(ciphertext []byte) (Locker, error)
}
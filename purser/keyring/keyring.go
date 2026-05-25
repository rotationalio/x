/*
Package keyring defines namespace-bound locker routing for purser.

Single-import clients use root aliases: [purser.Keyring], [purser.Registrator], [purser.Namespacer],
and [purser.Router] (see purser aliases.go).
*/
package keyring

import (
	"go.rtnl.ai/x/purser/keyring/keyspec"
	"go.rtnl.ai/x/purser/locker"
)

// Keyring is the full keyring: registration, namespace binding, and decrypt routing.
type Keyring interface {
	Registrator
	Namespacer
	Router
}

// Registrator registers lockers and manages the default write locker.
type Registrator interface {
	// Register adds a new locker derived from the given key specification and
	// returns it.
	Register(spec *keyspec.KeySpec) (locker.Locker, error)

	// Revoke disables or removes a locker by the given key ID.
	Revoke(keyID []byte) error

	// SetDefault sets the default locker to be used for new namespaces or
	// unbound operations.
	SetDefault(l locker.Locker) error
}

// Namespacer maps namespaces to lockers and resolves the locker used to seal a
// row.
type Namespacer interface {
	// Bind associates a locker with a given namespace.
	Bind(namespace string, l locker.Locker) error

	// Unbind removes the locker binding from the given namespace and returns
	// the locker that was bound.
	Unbind(namespace string) (locker.Locker, error)

	// Namespaces returns a snapshot of all namespace-to-keyID bindings.
	Namespaces() map[string][]byte

	// LockerFor returns the locker associated with the specified namespace.
	LockerFor(namespace string) (locker.Locker, error)
}

// Router parses wire metadata and looks up the locker that sealed a row.
type Router interface {
	// ParseKeyID extracts the key ID used for sealing from ciphertext metadata.
	ParseKeyID(ciphertext []byte) ([]byte, error)

	// Route locates the locker that matches the sealing key ID in the
	// ciphertext.
	Route(ciphertext []byte) (locker.Locker, error)
}

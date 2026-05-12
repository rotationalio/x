/*
Package vault defines the version-neutral [Vault] interface for opaque secret rows. Concrete
implementations live in versioned subpackages (for example [go.rtnl.ai/x/vault/v1]).
*/
package vault

import "context"

//=============================================================================
// Vault
//=============================================================================

// Vault is the contract for row operations (seal, open, compare-and-swap, move, delete).
// Versioned packages return an implementation from their constructor (for example [go.rtnl.ai/x/vault/v1.New]).
type Vault interface {
	Store(ctx context.Context, namespace string, plaintext []byte) (id string, err error)
	Retrieve(ctx context.Context, namespace, id string) (plaintext []byte, err error)
	Update(ctx context.Context, namespace, id string, plaintext []byte) error
	CompareAndSwap(ctx context.Context, namespace, id string, currentPlain, newPlain []byte) error
	MoveNamespace(ctx context.Context, oldNamespace, newNamespace, id string) error
	Delete(ctx context.Context, namespace, id string) error
}

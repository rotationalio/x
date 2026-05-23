package locker

// Registers v1 constructors with registry (locker/v1 does not import purser root).

import (
	"go.rtnl.ai/x/purser/locker/v1/constants"
	"go.rtnl.ai/x/purser/registry"
)

func init() {
	registry.Register(constants.Edition, registry.Hooks{
		FromSeed:     FromSeed,
		FromPassword: FromPassword,
		FromPKCS8:    FromPKCS8,
		FromKey:      FromKey,
		ParseKeyID:   ParseKeyID,
	})
}

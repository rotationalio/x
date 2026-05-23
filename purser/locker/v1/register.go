package locker

// Registers v1 constructors with registry (locker/v1 does not import purser root).

import (
	"go.rtnl.ai/x/purser/locker/v1/constants"
	"go.rtnl.ai/x/purser/registry"
)

func init() {
	if err := registry.Register(constants.Edition, registry.Hooks{
		WireVersion:  constants.Version,
		FromSeed:     FromSeed,
		FromPassword: FromPassword,
		FromPKCS8:    FromPKCS8,
		FromKey:      FromKey,
		ParseKeyID:   ParseKeyID,
	}); err != nil {
		panic(err)
	}
}

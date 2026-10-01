package set_test

import (
	"testing"

	. "go.rtnl.ai/x/set"
)

func TestSyncSet(t *testing.T) {
	t.Run("Int", MakeIntTests(func(items []int) Container[int] {
		return NewSyncSet(items...)
	}))
}

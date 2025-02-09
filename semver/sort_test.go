package semver_test

import (
	"testing"

	"go.rtnl.ai/x/assert"
	. "go.rtnl.ai/x/semver"
)

func TestSort(t *testing.T) {
	t.Run("Random", func(t *testing.T) {
		vers := make([]Version, 0, 100)
		for i := 0; i < 100; i++ {
			vers = append(vers, randVersion())
		}

		Sort(vers)
		for i := 1; i < len(vers); i++ {
			assert.Equal(t, 1, vers[i].Compare(vers[i-1]))
		}
	})

	t.Run("Static", func(t *testing.T) {
		vers := []Version{
			MustParse("2.1.0-rc.beta.1"),
			MustParse("2.0.0"),
			MustParse("1.1.0"),
			MustParse("2.0.0"),
			MustParse("2.0.0-alpha.1"),
			MustParse("1.0.0"),
			MustParse("2.1.0"),
			MustParse("2.1.0-rc.alpha.1"),
			MustParse("2.1.1"),
			MustParse("1.0.1"),
			MustParse("2.1.0-rc.2"),
			MustParse("2.1.0-rc.alpha.2"),
		}

		expected := []Version{
			MustParse("1.0.0"),
			MustParse("1.0.1"),
			MustParse("1.1.0"),
			MustParse("2.0.0-alpha.1"),
			MustParse("2.0.0"),
			MustParse("2.0.0"),
			MustParse("2.1.0-rc.2"),
			MustParse("2.1.0-rc.alpha.1"),
			MustParse("2.1.0-rc.alpha.2"),
			MustParse("2.1.0-rc.beta.1"),
			MustParse("2.1.0"),
			MustParse("2.1.1"),
		}

		Sort(vers)
		for i, actual := range vers {
			assert.Equal(t, expected[i], actual)
		}
	})
}

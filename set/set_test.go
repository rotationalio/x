package set_test

import (
	"testing"

	. "github.com/kansaslabs/x/set"
	"github.com/stretchr/testify/require"
)

// Ensure the Set works as expected with string types
func TestStringSet(t *testing.T) {

	// Can create and add to a set
	set := New()
	set.Add("foo")
	require.True(t, set.Contains("foo"))
	require.False(t, set.Contains(("Foo")))

	// The set does not store duplicates
	set.Add("foo")
	require.Len(t, set, 1)

	// Sets can grow
	set.Add("bar")
	require.Len(t, set, 2)

	// Can remove an element
	require.NoError(t, set.Remove("foo"))
	require.False(t, set.Contains(("foo")))
	require.Len(t, set, 1)

	// Cannot remove an element that's not in a set
	require.EqualError(t, set.Remove("foo"), "foo is not a member of the set")

	// Can discard an element that's not in a set
	require.False(t, set.Contains(("foo")))
	set.Discard("foo")
	require.Len(t, set, 1)

	// Can discard an element that is in a set
	set.Discard("bar")
	require.Len(t, set, 0)
	require.False(t, set.Contains(("bar")))
}

// Ensure the Set is a map and can be used with standard map operations
func TestSetIsMap(t *testing.T) {
	set := make(Set)
	set["foo"] = struct{}{}
	require.Len(t, set, 1)

	_, ok := set["foo"]
	require.True(t, ok)

	delete(set, "foo")
	require.Len(t, set, 0)
	_, ok = set["foo"]
	require.False(t, ok)
}

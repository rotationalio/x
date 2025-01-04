package set_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	. "go.rtnl.ai/x/set"
)

// Ensure the Set works as expected with string types
func TestStringSet(t *testing.T) {
	// Can create and add to a set
	set := New()
	set.Add("foo")
	require.True(t, set.Contains("foo"))
	require.False(t, set.Contains("Foo"))

	// The set does not store duplicates
	set.Add("foo")
	require.Len(t, set, 1)

	// Sets can grow
	set.Add("bar")
	require.Len(t, set, 2)

	// Can remove an element
	require.NoError(t, set.Remove("foo"))
	require.False(t, set.Contains("foo"))
	require.Len(t, set, 1)

	// Cannot remove an element that's not in a set
	require.EqualError(t, set.Remove("foo"), "foo is not a member of the set")

	// Can discard an element that's not in a set
	require.False(t, set.Contains("foo"))
	set.Discard("foo")
	require.Len(t, set, 1)

	// Can discard an element that is in a set
	set.Discard("bar")
	require.Len(t, set, 0)
	require.False(t, set.Contains("bar"))
}

// Test MultiSet Pop and Clear
func TestMultiStringSet(t *testing.T) {
	values := []string{"foo", "red", "bar", "try"}

	// Should be able to create a set with multiple items
	set := New("foo", "red", "bar", "try")
	require.Len(t, set, 4)

	val, err := set.Pop()
	require.NoError(t, err)
	require.NotContains(t, set, val)
	require.Contains(t, values, val)
	require.Len(t, set, 3)

	set.Clear()
	require.Len(t, set, 0)

	_, err = set.Pop()
	require.EqualError(t, err, ErrEmptySet.Error())
}

func TestNullEmptySets(t *testing.T) {
	require.True(t, IsNull(nil))
	require.True(t, IsNull(New()))
	require.True(t, New().IsEmpty())

	require.False(t, IsNull(New("a", "b", "c")))
	require.False(t, New("a", "b", "c").IsEmpty())
}

func TestSetComparisons(t *testing.T) {
	a := New("a", "b", "c", "d", "e")
	b := New("f", "g", "h", "i", "j")
	c := New("d", "e")

	require.True(t, a.IsDisjoint(b))
	require.True(t, b.IsDisjoint(a))
	require.False(t, a.IsDisjoint(c))
	require.False(t, c.IsDisjoint(a))

	require.True(t, c.IsSubset(a))
	require.False(t, c.IsSubset(b))
	require.False(t, a.IsSubset(c))
	require.False(t, a.IsSubset(b))

	require.True(t, a.IsSuperset(c))
	require.False(t, a.IsSuperset(b))
	require.False(t, c.IsSuperset(a))
}

func TestSetMath(t *testing.T) {
	a := New("a", "b", "c", "d", "e")
	b := New("f", "g", "h", "i", "j")
	c := New("d", "e", "f", "g")
	d := New("d")

	abc := a.Union(b, c)
	require.Len(t, abc, 10)

	require.True(t, a.Intersection(b, c).IsEmpty())

	acd := a.Intersection(c, d)
	require.Len(t, acd, 1)
	require.Contains(t, acd, "d")

	ac := a.Difference(c, b)
	require.Len(t, ac, 3)
	require.Contains(t, ac, "a")
	require.Contains(t, ac, "b")
	require.Contains(t, ac, "c")

	acs := a.SymmetricDifference(c)
	require.Len(t, acs, 5)
	require.Contains(t, acs, "a")
	require.Contains(t, acs, "b")
	require.Contains(t, acs, "c")
	require.Contains(t, acs, "f")
	require.Contains(t, acs, "g")

}

// Ensure the Set is a map and can be used with standard map operations
func TestSetIsMap(t *testing.T) {
	// Can directly make a set
	set := make(Set)

	// can directly add to a set and len() it
	set["foo"] = struct{}{}
	require.Len(t, set, 1)

	// can directly query if elements are in a set
	_, ok := set["foo"]
	require.True(t, ok)

	// can directly delete elements in a set
	delete(set, "foo")
	require.Len(t, set, 0)
	_, ok = set["foo"]
	require.False(t, ok)

	// can range over the elements in a set
	set.Add("foo")
	set.Add("bar")
	set.Add("baz")
	for key := range set {
		require.Contains(t, set, key)
	}
}

// Ensure the Set works as expected with int types
func TestIntSet(t *testing.T) {
	// Can create and add to a set
	set := New()
	set.Add(42)
	require.True(t, set.Contains(42))
	require.False(t, set.Contains(24))

	// The set does not store duplicates
	set.Add(42)
	require.Len(t, set, 1)

	// Sets can grow
	set.Add(7)
	require.Len(t, set, 2)

	// Can remove an element
	require.NoError(t, set.Remove(42))
	require.False(t, set.Contains(42))
	require.Len(t, set, 1)

	// Cannot remove an element that's not in a set
	require.EqualError(t, set.Remove(42), "42 is not a member of the set")

	// Can discard an element that's not in a set
	require.False(t, set.Contains(42))
	set.Discard(42)
	require.Len(t, set, 1)

	// Can discard an element that is in a set
	set.Discard(7)
	require.Len(t, set, 0)
	require.False(t, set.Contains(7))
}

// Test the copy operation
func TestCopy(t *testing.T) {
	s := New(1, 2, 3, 4, 5, 6)
	r := s.Copy()
	require.Equal(t, s, r)

	s.Clear()
	require.Len(t, s, 0)
	require.Len(t, r, 6)
}

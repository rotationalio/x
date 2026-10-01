package set_test

import (
	"fmt"
	"testing"

	"go.rtnl.ai/x/assert"
	. "go.rtnl.ai/x/set"
)

func TestHashSet(t *testing.T) {
	t.Run("Color", MakeColorTests(func(items []Color) Container[Color] {
		return NewHashSet(items...)
	}))
}

// Color is three uint8 values for red, green, and blue.
type Color struct {
	R, G, B uint8
}

// Hash returns a six character string which is the concatenation of the two
// hexadecimal digits for the red, green, and blue values.
func (c Color) Hash() string {
	return fmt.Sprintf("%02x%02x%02x", c.R, c.G, c.B)
}

func MakeColorTests(mk func([]Color) Container[Color]) func(t *testing.T) {
	// Test sets: first and second are disjoint, third is half of the
	// elements from the first and half from the second.
	var first, second []Color
	for i := 1; i <= 50; i++ {
		first = append(first, Color{R: uint8(i)})
		second = append(second, Color{G: uint8(i)})
	}
	third := append(append([]Color{}, first[:25]...), second[:25]...)

	white := Color{R: 255, G: 255, B: 255}

	return func(t *testing.T) {
		t.Run("Add", func(t *testing.T) {
			s := mk([]Color{})
			for i, v := range first {
				assert.False(t, s.Contains(v))
				assert.True(t, s.Add(v))
				assert.True(t, s.Contains(v))
				assert.Equal(t, i+1, s.Size())
			}
			assert.Equal(t, len(first), s.Size())
		})

		t.Run("AddNoModify", func(t *testing.T) {
			s := mk(first)
			for _, v := range first {
				assert.False(t, s.Add(v))
			}
			assert.Equal(t, len(first), s.Size())
		})

		t.Run("Remove", func(t *testing.T) {
			s := mk(third)
			for i, v := range third {
				assert.True(t, s.Contains(v))
				assert.True(t, s.Remove(v))
				assert.False(t, s.Contains(v))
				assert.Equal(t, len(third)-i-1, s.Size())
			}
			assert.Equal(t, 0, s.Size())
		})

		t.Run("RemoveNoModify", func(t *testing.T) {
			s := mk(second)
			for _, v := range first {
				assert.False(t, s.Remove(v))
				assert.Equal(t, len(second), s.Size())
			}
			assert.Equal(t, len(second), s.Size())
		})

		t.Run("Update", func(t *testing.T) {
			s := mk(second)
			assert.True(t, s.Update(mk(first)))
			assert.Equal(t, len(second)+len(first), s.Size())
			assert.True(t, s.Contains(second...))
			assert.True(t, s.Contains(first...))
		})

		t.Run("UpdateNoModify", func(t *testing.T) {
			s := mk(first)
			assert.False(t, s.Update(mk(first)))
			assert.Equal(t, len(first), s.Size())
			assert.True(t, s.Contains(first...))
		})

		t.Run("Empty", func(t *testing.T) {
			s := mk([]Color{})
			assert.True(t, s.Empty())
			assert.Equal(t, 0, s.Size())

			assert.True(t, s.Add(white))
			assert.False(t, s.Empty())
			assert.Equal(t, 1, s.Size())
		})

		t.Run("Clear", func(t *testing.T) {
			s := mk(first)
			assert.Equal(t, len(first), s.Size())

			s.Clear()
			assert.True(t, s.Empty())
		})

		t.Run("Copy", func(t *testing.T) {
			s := mk(first)
			c := s.Copy()
			assert.True(t, s.Equal(c))

			// Modifying s should not modify c
			assert.True(t, s.Add(white))
			assert.False(t, c.Contains(white))
		})

		t.Run("Disjoint", func(t *testing.T) {
			s := mk(first)
			assert.True(t, s.Disjoint(mk(second)))
			assert.False(t, s.Disjoint(mk(third)))
		})

		t.Run("Equal", func(t *testing.T) {
			s := mk(first)
			assert.True(t, s.Equal(mk(first)))
			assert.False(t, s.Equal(mk(second)))
		})

		t.Run("Subset", func(t *testing.T) {
			firsts := mk(first)
			s := mk(first[:6])

			assert.False(t, firsts.Subset(s))
			assert.True(t, s.Subset(firsts))

			s.Add(second[:6]...)
			assert.False(t, firsts.Subset(s))
			assert.False(t, s.Subset(firsts))
		})

		t.Run("Superset", func(t *testing.T) {
			firsts := mk(first)
			s := mk(first[:6])

			assert.False(t, s.Superset(firsts))
			assert.True(t, firsts.Superset(s))

			s.Add(second[:6]...)
			assert.False(t, s.Superset(firsts))
			assert.False(t, firsts.Superset(s))
		})

		t.Run("Union", func(t *testing.T) {
			s := mk(first)
			o := mk(second)

			u := s.Union(o)
			assert.False(t, u.Disjoint(s))
			assert.False(t, u.Disjoint(o))
			assert.Equal(t, len(first)+len(second), u.Size())
		})

		t.Run("Intersection", func(t *testing.T) {
			s := mk(first)
			o := mk(second)
			r := mk(third)

			i := s.Intersection(o)
			assert.Equal(t, 0, i.Size())

			// Ensure everything in j is in both s and r (the half of
			// third drawn from the first).
			j := s.Intersection(r)
			assert.True(t, j.Subset(s))
			assert.True(t, j.Subset(r))
			for v := range j.Items() {
				assert.True(t, s.Contains(v))
				assert.True(t, r.Contains(v))
			}

		})

		t.Run("Difference", func(t *testing.T) {
			s := mk(first)
			o := mk(second)
			r := mk(third)

			// first - second = first
			d := s.Difference(o)
			assert.True(t, d.Equal(s))

			// first - third = half of first not in third
			e := s.Difference(r)
			assert.True(t, e.Subset(s))
			assert.False(t, e.Subset(o))
		})

		t.Run("SymmetricDifference", func(t *testing.T) {
			s := mk(first)
			o := mk(second)
			r := mk(third)

			// first ^ second = first + second
			d := s.SymmetricDifference(o)
			assert.Equal(t, len(first)+len(second), d.Size())
			assert.True(t, d.Superset(s))
			assert.True(t, d.Superset(o))

			// e holds elements in exactly one of r and s.
			e := r.SymmetricDifference(s)
			for v := range e.Items() {
				assert.True(t, r.Contains(v) != s.Contains(v))
			}
		})

		t.Run("Slice", func(t *testing.T) {
			s := mk(first)
			u := s.Slice()

			// Simple equality will not work since some sets do not
			// return the elements in the original order.
			assert.Equal(t, len(first), len(u))
			assert.ElementsMatch(t, first, u)
		})

		t.Run("Items", func(t *testing.T) {
			s := mk(first)
			for v := range s.Items() {
				assert.True(t, s.Contains(v))
			}
		})
	}
}

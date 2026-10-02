package set_test

import (
	"encoding/json"
	"testing"

	"go.rtnl.ai/x/assert"
	. "go.rtnl.ai/x/set"
)

func TestSet(t *testing.T) {
	t.Run("Int", MakeIntTests(func(items []int) Container[int] {
		return New(items...)
	}))
}

func MakeIntTests(mk func([]int) Container[int]) func(t *testing.T) {
	// Test sets
	evens := []int{2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 30, 32, 34, 36, 38, 40, 42, 44, 46, 48, 50, 52, 54, 56, 58, 60, 62, 64, 66, 68, 70, 72, 74, 76, 78, 80, 82, 84, 86, 88, 90, 92, 94, 96, 98, 100}
	odds := []int{1, 3, 5, 7, 9, 11, 13, 15, 17, 19, 21, 23, 25, 27, 29, 31, 33, 35, 37, 39, 41, 43, 45, 47, 49, 51, 53, 55, 57, 59, 61, 63, 65, 67, 69, 71, 73, 75, 77, 79, 81, 83, 85, 87, 89, 91, 93, 95, 97, 99}
	trips := []int{3, 6, 9, 12, 15, 18, 21, 24, 27, 30, 33, 36, 39, 42, 45, 48, 51, 54, 57, 60, 63, 66, 69, 72, 75, 78, 81, 84, 87, 90, 93, 96, 99}

	return func(t *testing.T) {
		t.Run("Add", func(t *testing.T) {
			s := mk([]int{})
			for i, v := range evens {
				assert.False(t, s.Contains(v))
				assert.True(t, s.Add(v))
				assert.True(t, s.Contains(v))
				assert.Equal(t, i+1, s.Size())
			}
			assert.Equal(t, len(evens), s.Size())
		})

		t.Run("AddNoModify", func(t *testing.T) {
			s := mk(evens)
			for _, v := range evens {
				assert.False(t, s.Add(v))
			}
			assert.Equal(t, len(evens), s.Size())
		})

		t.Run("Remove", func(t *testing.T) {
			s := mk(trips)
			for i, v := range trips {
				assert.True(t, s.Contains(v))
				assert.True(t, s.Remove(v))
				assert.False(t, s.Contains(v))
				assert.Equal(t, len(trips)-i-1, s.Size())
			}
			assert.Equal(t, 0, s.Size())
		})

		t.Run("RemoveNoModify", func(t *testing.T) {
			s := mk(odds)
			for _, v := range evens {
				assert.False(t, s.Remove(v))
				assert.Equal(t, len(odds), s.Size())
			}
			assert.Equal(t, len(odds), s.Size())
		})

		t.Run("Update", func(t *testing.T) {
			s := mk(odds)
			assert.True(t, s.Update(mk(evens)))
			assert.Equal(t, len(odds)+len(evens), s.Size())
			assert.True(t, s.Contains(odds...))
			assert.True(t, s.Contains(evens...))
		})

		t.Run("UpdateNoModify", func(t *testing.T) {
			s := mk(evens)
			assert.False(t, s.Update(mk(evens)))
			assert.Equal(t, len(evens), s.Size())
			assert.True(t, s.Contains(evens...))
		})

		t.Run("Empty", func(t *testing.T) {
			s := mk([]int{})
			assert.True(t, s.Empty())
			assert.Equal(t, 0, s.Size())

			assert.True(t, s.Add(1))
			assert.False(t, s.Empty())
			assert.Equal(t, 1, s.Size())
		})

		t.Run("Clear", func(t *testing.T) {
			s := mk(evens)
			assert.Equal(t, len(evens), s.Size())

			s.Clear()
			assert.True(t, s.Empty())
		})

		t.Run("Copy", func(t *testing.T) {
			s := mk(evens)
			c := s.Copy()
			assert.Equal(t, s, c)

			// Modifying s should not modify c
			assert.True(t, s.Add(1024))
			assert.False(t, c.Contains(1024))
		})

		t.Run("Disjoint", func(t *testing.T) {
			s := mk(evens)
			assert.True(t, s.Disjoint(mk(odds)))
			assert.False(t, s.Disjoint(mk(trips)))
		})

		t.Run("Equal", func(t *testing.T) {
			s := mk(evens)
			assert.True(t, s.Equal(mk(evens)))
			assert.False(t, s.Equal(mk(odds)))
		})

		t.Run("Subset", func(t *testing.T) {
			evens := mk(evens)
			s := mk([]int{20, 22, 24, 26, 28, 30})

			assert.False(t, evens.Subset(s))
			assert.True(t, s.Subset(evens))

			s.Add(21, 23, 25, 27, 29, 31)
			assert.False(t, evens.Subset(s))
			assert.False(t, s.Subset(evens))
		})

		t.Run("Superset", func(t *testing.T) {
			evens := mk(evens)
			s := mk([]int{20, 22, 24, 26, 28, 30})

			assert.False(t, s.Superset(evens))
			assert.True(t, evens.Superset(s))

			s.Add(21, 23, 25, 27, 29, 31)
			assert.False(t, s.Superset(evens))
			assert.False(t, evens.Superset(s))
		})

		t.Run("Union", func(t *testing.T) {
			s := mk(evens)
			o := mk(odds)

			u := s.Union(o)
			assert.False(t, u.Disjoint(s))
			assert.False(t, u.Disjoint(o))
			assert.Equal(t, len(evens)+len(odds), u.Size())
		})

		t.Run("Intersection", func(t *testing.T) {
			s := mk(evens)
			o := mk(odds)
			r := mk(trips)

			i := s.Intersection(o)
			assert.Equal(t, 0, i.Size())

			// Ensure everything in j is even and divisible by three
			j := s.Intersection(r)
			assert.True(t, j.Subset(s))
			assert.True(t, j.Subset(r))
			for v := range j.Items() {
				assert.True(t, s.Contains(v))
				assert.True(t, r.Contains(v))
			}

		})

		t.Run("Difference", func(t *testing.T) {
			s := mk(evens)
			o := mk(odds)
			r := mk(trips)

			// evens - odds = evens
			d := s.Difference(o)
			assert.True(t, d.Equal(s))

			// evens - trips = evens divisible by 3
			e := s.Difference(r)
			assert.True(t, e.Subset(s))
			assert.False(t, e.Subset(o))
		})

		t.Run("SymmetricDifference", func(t *testing.T) {
			s := mk(evens)
			o := mk(odds)
			r := mk(trips)

			// evens ^ odds = evens + odds
			d := s.SymmetricDifference(o)
			assert.Equal(t, len(evens)+len(odds), d.Size())
			assert.True(t, d.Superset(s))
			assert.True(t, d.Superset(o))

			// e will be divisible by 3 and odd or even and not
			// divisible by 3.
			e := r.SymmetricDifference(s)
			for v := range e.Items() {
				if v%2 == 0 {
					assert.True(t, v%3 != 0)
				} else {
					assert.True(t, v%3 == 0)
				}
			}
		})

		t.Run("Slice", func(t *testing.T) {
			s := mk(evens)
			u := s.Slice()

			// Simple equality will not work since some sets do not
			// return the elements in the original order.
			assert.Equal(t, len(evens), len(u))
			assert.ElementsMatch(t, evens, u)
		})

		t.Run("Items", func(t *testing.T) {
			s := mk(evens)
			for v := range s.Items() {
				assert.True(t, s.Contains(v))
			}
		})

		t.Run("JSON", func(t *testing.T) {
			e := mk(evens)
			data, err := json.Marshal(e)
			assert.Ok(t, err)

			f := mk([]int{})
			assert.Ok(t, json.Unmarshal(data, f))
			assert.True(t, e.Equal(f))
		})
	}
}

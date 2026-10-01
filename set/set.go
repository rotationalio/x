package set

import "iter"

type empty struct{}

var sentinel = empty{}

// Set is a generic implementation of the set mathematical data structure. It is
// optimized for convenience of use, readability, and correctness to replace
// map[any]struct{} in code.
//
// This set is not thread-safe and should not be used concurrently. Use SyncSet which
// provides a thread-safe implementation of the set data structure.
type Set[T comparable] struct {
	items map[T]empty
}

// Ensure that Set implements the Container interface.
var _ Container[int] = (*Set[int])(nil)

// New creates a Set with the specified elements.
func New[T comparable](items ...T) *Set[T] {
	s := Make[T](len(items))
	s.Add(items...)
	return s
}

// Make creates a Set with initial underlying capacity of size.
//
// Sets automatically grow or shrink as items are added or removed similar to maps.
// T should be any comparable type, but note that the Set uses shallow equality for
// comparision, for deep equality use HashSet.
func Make[T comparable](size int) *Set[T] {
	return &Set[T]{
		items: make(map[T]empty, max(0, size)),
	}
}

// Adds the specified elements to the set.
func (s *Set[T]) Add(items ...T) bool {
	if s.items == nil && len(items) > 0 {
		s.items = make(map[T]empty, len(items))
	}

	modified := false
	for _, item := range items {
		if _, ok := s.items[item]; !ok {
			s.items[item] = sentinel
			modified = true
		}
	}
	return modified
}

// Removes the specified elements from the set.
func (s *Set[T]) Remove(items ...T) bool {
	if len(s.items) == 0 {
		return false
	}

	modified := false
	for _, item := range items {
		if _, ok := s.items[item]; ok {
			delete(s.items, item)
			modified = true
		}
	}
	return modified
}

// Returns true if the set contains all of the specified elements.
func (s *Set[T]) Contains(items ...T) bool {
	if len(s.items) == 0 {
		return false
	}

	for _, item := range items {
		if _, ok := s.items[item]; !ok {
			return false
		}
	}
	return true
}

// Update the set with the elements of another set.
func (s *Set[T]) Update(src Container[T]) bool {
	return update(src, s)
}

// Returns the number of elements in the set.
func (s *Set[T]) Size() int {
	return len(s.items)
}

// Returns true if the set is empty.
func (s *Set[T]) Empty() bool {
	return len(s.items) == 0
}

// Removes all elements from the set.
func (s *Set[T]) Clear() {
	s.items = nil
}

// Returns a new set with the elements of the set.
func (s *Set[T]) Copy() Container[T] {
	clone := Make[T](len(s.items))
	update(s, clone)
	return clone
}

// Returns true if the set is disjoint with another set.
// Two sets are disjoint if they have no elements in common.
func (s *Set[T]) Disjoint(other Container[T]) bool {
	return disjoint(s, other)
}

// Returns true if the set is equal to another set.
// Two sets are equal if they contain the same elements and are the same size.
func (s *Set[T]) Equal(other Container[T]) bool {
	return equal(s, other)
}

// Returns true if the s is a subset of other.
// s is a subset of other if it is smaller than other and all of its elements are also in other.
func (s *Set[T]) Subset(other Container[T]) bool {
	return subset(s, other)
}

// Returns true if the s is a superset of other.
// s is a superset of other if other is a subset of s.
func (s *Set[T]) Superset(other Container[T]) bool {
	return superset(s, other)
}

// Returns the union of the set and another set.
// The union of two sets is the set of all elements that are in either set.
func (s *Set[T]) Union(other Container[T]) Container[T] {
	size := max(s.Size(), other.Size())
	dst := Make[T](size)
	dst.Update(s)
	dst.Update(other)
	return dst
}

// Returns the intersection of the set and another set.
// The intersection of two sets is the set of all elements that are in both sets.
func (s *Set[T]) Intersection(other Container[T]) Container[T] {
	dst := Make[T](0)
	intersection(dst, s, other)
	return dst
}

// Returns the difference of the set and another set.
// The difference of two sets is the set of all elements that are in the first set but not in the second.
func (s *Set[T]) Difference(other Container[T]) Container[T] {
	dst := Make[T](max(0, s.Size()-other.Size()))
	difference(dst, s, other)
	return dst
}

// Returns the symmetric difference of the set and another set.
// The symmetric difference of two sets is the set of all elements that are in either set but not in both.
func (s *Set[T]) SymmetricDifference(other Container[T]) Container[T] {
	dst := Make[T](max(0, s.Size()-other.Size()))
	symmetricDifference(dst, s, other)
	return dst
}

// Returns a slice of the elements in the set.
func (s *Set[T]) Slice() []T {
	out := make([]T, 0, len(s.items))
	for item := range s.items {
		out = append(out, item)
	}
	return out
}

// Returns an iterator for the elements in the set.
func (s *Set[T]) Items() iter.Seq[T] {
	return func(yield func(T) bool) {
		for item := range s.items {
			if !yield(item) {
				return
			}
		}
	}
}

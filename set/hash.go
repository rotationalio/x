package set

import "iter"

// Implements the Container interface as a mathematical set, but rather than using
// shallow equality, it uses a hash function to determine equality. This type of set is
// useful for sets of complex types with nested fields that require deep equality.
type HashSet[T Hashable[H], H Hash] struct {
	hfn   HashFunc[T, H]
	items map[H]T
}

//============================================================================
// Hash Interface
//============================================================================

// The output type of a Hash() function defined on a type.
type Hash interface {
	~string | ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// A type that can be hashed to a Hash value.
type Hashable[H Hash] interface {
	Hash() H
}

// A function that can be used to hash a type to a Hash value.
type HashFunc[T any, H Hash] func(T) H

// Creates a closure around the type to use as a repeatable hash function.
func Hasher[T Hashable[H], H Hash]() HashFunc[T, H] {
	return func(item T) H {
		return item.Hash()
	}
}

//============================================================================
// HashSet Implementation
//============================================================================

// NewHashSet creates a new HashSet with the specified items.
func NewHashSet[T Hashable[H], H Hash](items ...T) *HashSet[T, H] {
	set := &HashSet[T, H]{
		hfn:   Hasher[T](),
		items: make(map[H]T),
	}

	set.Add(items...)
	return set
}

// MakeHashSet creates a new HashSet with the specified initial capacity.
func MakeHashSet[T Hashable[H], H Hash](capacity int) *HashSet[T, H] {
	return &HashSet[T, H]{
		hfn:   Hasher[T](),
		items: make(map[H]T, max(0, capacity)),
	}
}

// Adds the specified items to the HashSet.
func (s *HashSet[T, H]) Add(items ...T) bool {
	if s.items == nil && len(items) > 0 {
		s.items = make(map[H]T, len(items))
	}

	modified := false
	for _, item := range items {
		key := s.hfn(item)
		if _, ok := s.items[key]; !ok {
			s.items[key] = item
			modified = true
		}
	}
	return modified
}

// Removes the specified items from the HashSet.
func (s *HashSet[T, H]) Remove(items ...T) bool {
	if len(s.items) == 0 {
		return false
	}

	modified := false
	for _, item := range items {
		key := s.hfn(item)
		if _, ok := s.items[key]; ok {
			delete(s.items, key)
			modified = true
		}
	}
	return modified
}

// Returns true if the HashSet contains all of the specified items.
func (s *HashSet[T, H]) Contains(items ...T) bool {
	if len(s.items) == 0 {
		return false
	}

	for _, item := range items {
		key := s.hfn(item)
		if _, ok := s.items[key]; !ok {
			return false
		}
	}
	return true
}

// Updates the HashSet with the elements of another set.
func (s *HashSet[T, H]) Update(src Container[T]) bool {
	return update(src, s)
}

// Returns the number of elements in the HashSet.
func (s *HashSet[T, H]) Size() int {
	return len(s.items)
}

// Returns true if the HashSet is empty.
func (s *HashSet[T, H]) Empty() bool {
	return len(s.items) == 0
}

// Removes all elements from the HashSet.
func (s *HashSet[T, H]) Clear() {
	s.items = nil
}

// Returns a new HashSet with the elements of the HashSet.
func (s *HashSet[T, H]) Copy() Container[T] {
	clone := MakeHashSet[T](len(s.items))
	update(s, clone)
	return clone
}

// Returns true if the HashSet is disjoint with another set.
// Two sets are disjoint if they have no elements in common.
func (s *HashSet[T, H]) Disjoint(other Container[T]) bool {
	return disjoint(s, other)
}

// Returns true if the HashSet is equal to another set.
// Two sets are equal if they contain the same elements and are the same size.
func (s *HashSet[T, H]) Equal(other Container[T]) bool {
	return equal(s, other)
}

// Returns true if the HashSet is a subset of another set.
// A set is a subset of another set if all of its elements are also in the other set.
func (s *HashSet[T, H]) Subset(other Container[T]) bool {
	return subset(s, other)
}

// Returns true if the HashSet is a superset of another set.
// A set is a superset of another set if all of its elements are also in the other set.
func (s *HashSet[T, H]) Superset(other Container[T]) bool {
	return superset(s, other)
}

// Returns the union of the HashSet and another set.
// The union of two sets is the set of all elements that are in either set.
func (s *HashSet[T, H]) Union(other Container[T]) Container[T] {
	size := max(s.Size(), other.Size())
	dst := MakeHashSet[T](size)
	dst.Update(s)
	dst.Update(other)
	return dst
}

// Returns the intersection of the HashSet and another set.
// The intersection of two sets is the set of all elements that are in both sets.
func (s *HashSet[T, H]) Intersection(other Container[T]) Container[T] {
	dst := MakeHashSet[T](0)
	intersection(dst, s, other)
	return dst
}

// Returns the difference of the HashSet and another set.
// The difference of two sets is the set of all elements that are in the HashSet but not in the other set.
func (s *HashSet[T, H]) Difference(other Container[T]) Container[T] {
	dst := MakeHashSet[T](max(0, s.Size()-other.Size()))
	difference(dst, s, other)
	return dst
}

// Returns the symmetric difference of the HashSet and another set.
// The symmetric difference of two sets is the set of all elements that are in either set but not in both.
func (s *HashSet[T, H]) SymmetricDifference(other Container[T]) Container[T] {
	dst := MakeHashSet[T](max(0, s.Size()-other.Size()))
	symmetricDifference(dst, s, other)
	return dst
}

// Returns a slice of the elements in the HashSet.
func (s *HashSet[T, H]) Slice() []T {
	out := make([]T, 0, len(s.items))
	for _, item := range s.items {
		out = append(out, item)
	}
	return out
}

// Returns an iterator for the elements in the HashSet.
func (s *HashSet[T, H]) Items() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, item := range s.items {
			if !yield(item) {
				return
			}
		}
	}
}

//============================================================================
// Serialization
//============================================================================

func (s *HashSet[T, H]) MarshalJSON() ([]byte, error) {
	return marshalJSON(s)
}

func (s *HashSet[T, H]) UnmarshalJSON(data []byte) error {
	return unmarshalJSON(data, s)
}

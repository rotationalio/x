package set

import (
	"iter"
	"sync"
)

type SyncSet[T comparable] struct {
	mu  sync.RWMutex
	set *Set[T]
}

// Ensure that SyncSet implements the Container interface.
var _ Container[int] = (*SyncSet[int])(nil)

// NewSyncSet creates a new SyncSet with the specified elements.
func NewSyncSet[T comparable](items ...T) *SyncSet[T] {
	return &SyncSet[T]{
		set: New(items...),
	}
}

// MakeSyncSet creates a new SyncSet with the specified initial capacity.
func MakeSyncSet[T comparable](size int) *SyncSet[T] {
	return &SyncSet[T]{
		set: Make[T](size),
	}
}

// Adds the specified elements to the set.
func (s *SyncSet[T]) Add(items ...T) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.set == nil {
		s.set = New(items...)
		return true
	}
	return s.set.Add(items...)
}

// Removes the specified elements from the set.
func (s *SyncSet[T]) Remove(items ...T) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.set == nil {
		return false
	}
	return s.set.Remove(items...)
}

// Returns true if the set contains all of the specified elements.
func (s *SyncSet[T]) Contains(items ...T) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.set == nil {
		return false
	}
	return s.set.Contains(items...)
}

// Update the set with the elements of another set.
// NOTE: this is only safe if the src container is a SyncSet.
func (s *SyncSet[T]) Update(src Container[T]) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.set == nil {
		s.set = New[T]()
	}

	if safe, ok := src.(*SyncSet[T]); ok {
		safe.mu.RLock()
		defer safe.mu.RUnlock()
		return s.set.Update(safe.set)
	}

	return s.set.Update(src)
}

// Returns the number of elements in the set.
func (s *SyncSet[T]) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.set == nil {
		return 0
	}
	return s.set.Size()
}

// Returns true if the set is empty.
func (s *SyncSet[T]) Empty() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.set == nil {
		return true
	}
	return s.set.Empty()
}

// Removes all elements from the set.
func (s *SyncSet[T]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.set == nil {
		return
	}
	s.set.Clear()
}

// Returns a new set with the elements of the set.
func (s *SyncSet[T]) Copy() Container[T] {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clone := &SyncSet[T]{}

	if s.set != nil {
		clone.set = s.set.Copy().(*Set[T])
	}
	return clone
}

// Returns true if the set is disjoint with another set.
// NOTE: this is only safe if the other container is a SyncSet.
func (s *SyncSet[T]) Disjoint(other Container[T]) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.set == nil {
		s.set = New[T]()
	}

	if safe, ok := other.(*SyncSet[T]); ok {
		safe.mu.RLock()
		defer safe.mu.RUnlock()
		return s.set.Disjoint(safe.set)
	}

	return s.set.Disjoint(other)
}

// Returns true if the set is equal to another set.
// NOTE: this is only safe if the other container is a SyncSet.
func (s *SyncSet[T]) Equal(other Container[T]) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.set == nil {
		return other == nil || other.Empty()
	}

	if safe, ok := other.(*SyncSet[T]); ok {
		safe.mu.RLock()
		defer safe.mu.RUnlock()
		return s.set.Equal(safe.set)
	}

	return s.set.Equal(other)
}

// Returns true if the other is a subset of s.
// NOTE: this is only safe if the other container is a SyncSet.
func (s *SyncSet[T]) Subset(other Container[T]) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.set == nil {
		s.set = New[T]()
	}

	if safe, ok := other.(*SyncSet[T]); ok {
		safe.mu.RLock()
		defer safe.mu.RUnlock()
		return s.set.Subset(safe.set)
	}

	return s.set.Subset(other)
}

// Returns true if the other is a superset of s.
// NOTE: this is only safe if the other container is a SyncSet.
func (s *SyncSet[T]) Superset(other Container[T]) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.set == nil {
		s.set = New[T]()
	}

	if safe, ok := other.(*SyncSet[T]); ok {
		safe.mu.RLock()
		defer safe.mu.RUnlock()
		return s.set.Superset(safe.set)
	}

	return s.set.Superset(other)
}

// Returns the union of the set and another set.
// NOTE: this is only safe if the other container is a SyncSet.
func (s *SyncSet[T]) Union(other Container[T]) Container[T] {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.set == nil {
		return other.Copy()
	}

	if safe, ok := other.(*SyncSet[T]); ok {
		safe.mu.RLock()
		defer safe.mu.RUnlock()
		return s.set.Union(safe.set)
	}

	return s.set.Union(other)
}

// Returns the intersection of the set and another set.
// NOTE: this is only safe if the other container is a SyncSet.
func (s *SyncSet[T]) Intersection(other Container[T]) Container[T] {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.set == nil {
		s.set = New[T]()
	}

	if safe, ok := other.(*SyncSet[T]); ok {
		safe.mu.RLock()
		defer safe.mu.RUnlock()
		return s.set.Intersection(safe.set)
	}

	return s.set.Intersection(other)
}

// Returns the difference of the set and another set.
// NOTE: this is only safe if the other container is a SyncSet.
func (s *SyncSet[T]) Difference(other Container[T]) Container[T] {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.set == nil {
		s.set = New[T]()
	}

	if safe, ok := other.(*SyncSet[T]); ok {
		safe.mu.RLock()
		defer safe.mu.RUnlock()
		return s.set.Difference(safe.set)
	}

	return s.set.Difference(other)
}

// Returns the symmetric difference of the set and another set.
// NOTE: this is only safe if the other container is a SyncSet.
func (s *SyncSet[T]) SymmetricDifference(other Container[T]) Container[T] {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.set == nil {
		s.set = New[T]()
	}

	if safe, ok := other.(*SyncSet[T]); ok {
		safe.mu.RLock()
		defer safe.mu.RUnlock()
		return s.set.SymmetricDifference(safe.set)
	}

	return s.set.SymmetricDifference(other)
}

// Returns a slice of the elements in the set.
func (s *SyncSet[T]) Slice() []T {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.set == nil {
		return nil
	}

	return s.set.Slice()
}

// Returns an iterator for the elements in the set.
func (s *SyncSet[T]) Items() iter.Seq[T] {
	return func(yield func(T) bool) {
		s.mu.RLock()
		defer s.mu.RUnlock()

		if s.set == nil {
			return
		}

		for item := range s.set.Items() {
			if !yield(item) {
				return
			}
		}
	}
}

//============================================================================
// Serialization
//============================================================================

func (s *SyncSet[T]) MarshalJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.set == nil {
		return []byte("[]"), nil
	}
	return marshalJSON(s)
}

func (s *SyncSet[T]) UnmarshalJSON(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.set == nil {
		s.set = New[T]()
	}
	return unmarshalJSON(data, s.set)
}

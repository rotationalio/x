/*
Package set implements a `Set` and a `SyncSet` for common set operations. This is pretty
standard data structure implementation that uses a `map[interface{}]struct{}` and also
serves as a template for more specific set type implementations. The SyncSet provides
thread-safe concurrent access to a set using locking.
*/
package set

import (
	"errors"
	"fmt"
)

type void struct{}

var member void

// Errors that might be returned from various set operations.
var (
	ErrEmptySet = errors.New("empty sets do not support this operation")
)

// New creates and returns a new set, you can also use `make(Set)`.
func New(elems ...interface{}) (s Set) {
	s = make(Set)
	for _, elem := range elems {
		s.Add(elem)
	}
	return s
}

// Set is an unordered collection of distinct objects of any type. Common uses include
// membership testing, deduplication, and mathematical operations such as intersection,
// union, difference, and symmetric difference.
type Set map[interface{}]void

//===========================================================================
// Element Operations
//===========================================================================

// Add element to the set. Can also use `s[elem] = struct{}{}`.
func (s Set) Add(elem interface{}) {
	s[elem] = member
}

// Remove element from the set. Returns an error if elem is not contained in the set.
func (s Set) Remove(elem interface{}) error {
	if _, ok := s[elem]; !ok {
		return fmt.Errorf("%v is not a member of the set", elem)
	}
	delete(s, elem)
	return nil
}

// Discard element from the set if it is present. Can also use `delete(s, elem)`.
func (s Set) Discard(elem interface{}) {
	delete(s, elem)
}

// Pop removes and returns an arbitrary element from teh set. Returns an error if the
// set is empty. Note that it is far more efficient to range over the set then pop
// until empty. Also note that a type assertion is required to used the returned elem.
func (s Set) Pop() (interface{}, error) {
	for elem := range s {
		delete(s, elem)
		return elem, nil
	}
	return nil, ErrEmptySet
}

// Clear the set in place, removing all elements.
func (s Set) Clear() {
	for elem := range s {
		delete(s, elem)
	}
}

//===========================================================================
// Set Comparisons
//===========================================================================

// Contains returns true if the elem is included in the set.
func (s Set) Contains(elem interface{}) bool {
	_, ok := s[elem]
	return ok
}

// IsNull returns true iff the set has zero elements or is nil
func IsNull(s Set) bool {
	return s == nil || len(s) == 0
}

// IsEmpty returns true iff the set has zero elements
func (s Set) IsEmpty() bool {
	return len(s) == 0
}

// IsDisjoint returns true if the set has no elements in common with other. Sets are
// disjoint if and only if their intersection is the empty set.
func (s Set) IsDisjoint(other Set) bool {
	if len(other) < len(s) {
		// Loop through the smaller set to determine disjointedness
		return other.IsDisjoint(s)
	}

	for elem := range s {
		if other.Contains(elem) {
			return false
		}
	}

	return true
}

// IsSubset tests if every element in the set is in other.
func (s Set) IsSubset(other Set) bool {
	if len(s) > len(other) {
		// It can't be a subset if it has more items than the other
		return false
	}

	for elem := range s {
		if !other.Contains(elem) {
			return false
		}
	}

	return true
}

// IsSuperset tests if every element of other is in the set.
func (s Set) IsSuperset(other Set) bool {
	return other.IsSubset(s)
}

//===========================================================================
// Set Math
//===========================================================================

// Union returns a new set with elements from the set and all others.
func (s Set) Union(others ...Set) Set {
	r := s.Copy()
	for _, other := range others {
		for elem := range other {
			r.Add(elem)
		}
	}
	return r
}

// Intersection returns a new set with elements common to the set and all others.
func (s Set) Intersection(others ...Set) Set {
	// TODO: find the smallest set to perform the intersection on and benchmark if
	// that makes a difference in the performance of this computation.
	r := make(Set)
outer:
	for elem := range s {
		// Determine if the element is in all other sets otherwise skip it
		for _, other := range others {
			if !other.Contains(elem) {
				continue outer
			}
		}

		// Because we didn't continue outer, this elem is in all other sets
		r.Add(elem)
	}
	return r
}

// Difference returns a new set with elements in the set that are not in the others.
func (s Set) Difference(others ...Set) Set {
	r := make(Set)

outer:
	for elem := range s {
		// Determine if element is in any other set and if so, skip it
		for _, other := range others {
			if other.Contains(elem) {
				continue outer
			}
		}

		// Because we didn't continue outer, this elem is not in any other set
		r.Add(elem)
	}
	return r
}

// SymmetricDifference returns a new set with elements in either the set or other
// but not both. Note that this method can only accept a single input set.
func (s Set) SymmetricDifference(other Set) Set {
	r := make(Set)

	for elem := range s {
		if other.Contains(elem) {
			continue
		}
		r.Add(elem)
	}

	for elem := range other {
		if s.Contains(elem) {
			continue
		}
		r.Add(elem)
	}

	return r

}

// Copy returns a shallow copy of the set.
func (s Set) Copy() Set {
	r := make(Set)
	for elem := range s {
		r.Add(elem)
	}
	return r
}

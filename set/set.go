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
func New() Set {
	return make(Set)
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
	if len(s) == 0 {
		return nil, ErrEmptySet
	}

	for elem := range s {
		return elem, nil
	}
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

// IsDisjoint TODO: impelement
func (s Set) IsDisjoint(other Set) bool {
	return false
}

// IsSubset TODO: impelement
func (s Set) IsSubset(other Set) bool {
	return false
}

// IsSuperset TODO: impelement
func (s Set) IsSuperset(other Set) bool {
	return false
}

//===========================================================================
// Set Math
//===========================================================================

// Union TODO: impelement
func (s Set) Union(other Set) Set {
	return nil
}

// Intersection TODO: impelement
func (s Set) Intersection(other Set) Set {
	return nil
}

// Difference TODO: impelement
func (s Set) Difference(other Set) Set {
	return nil
}

// SymmetricDifference TODO: impelement
func (s Set) SymmetricDifference(other Set) Set {
	return nil
}

// Copy TODO: impelement
func (s Set) Copy(other Set) Set {
	return nil
}

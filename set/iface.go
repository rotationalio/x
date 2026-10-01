package set

import (
	"encoding/json"
	"iter"
)

// Container represents a common interface that all sets in this package must implement,
// it ensures that sets can be used interchangeably in the package and that external
// data types can be used with these sets.
type Container[T any] interface {
	// Add an element (or elements) to the container, returns true if the container is modified.
	Add(...T) bool

	// Remove an element (or elements) from the container, returns true if the container is modified.
	Remove(...T) bool

	// Returns true if the element is (or all elements are) a member of the container.
	Contains(...T) bool

	// Updates the container with the elements of another container, returns true if
	// the container was modified during the update operation.
	Update(Container[T]) bool

	// Returns the number of elements in the container.
	Size() int

	// Returns true if the container is empty (e.g. size == 0)
	Empty() bool

	// Removes all elements from the container.
	Clear()

	// Returns a new container with the elements of the container.
	Copy() Container[T]

	// Evaluates set disjointedness
	Disjoint(Container[T]) bool

	// Evaluates set equality
	Equal(Container[T]) bool

	// Evaluates subset relationship
	Subset(Container[T]) bool

	// Evaluates superset relationship
	Superset(Container[T]) bool

	// Computes the set union of the container and another container, returns a new container
	Union(Container[T]) Container[T]

	// Computes the set intersection of the container and another container, returns a new container
	Intersection(Container[T]) Container[T]

	// Computes the set difference of the container and another container, returns a new container
	Difference(Container[T]) Container[T]

	// Computes the set symmetric difference of the container and another container, returns a new container
	SymmetricDifference(Container[T]) Container[T]

	// Slice returns a slice of the elements in the container.
	Slice() []T

	// Items returns an interator for use with the range keyword.
	Items() iter.Seq[T]
}

//============================================================================
// Internal Functions Implemented on Container Interface
//============================================================================

// Update the destination container with the elements of the source container.
func update[T any](src, dst Container[T]) bool {
	modified := false
	for item := range src.Items() {
		if dst.Add(item) {
			modified = true
		}
	}
	return modified
}

// Compute the intersection of two containers and store the result in the destination container.
func intersection[T any](dst, a, b Container[T]) {
	var (
		smaller = a
		larger  = b
	)
	if a.Size() > b.Size() {
		smaller = b
		larger = a
	}
	for item := range smaller.Items() {
		if larger.Contains(item) {
			dst.Add(item)
		}
	}
}

// Computes the difference of two containers and stores the result in the destination container.
func difference[T any](dst, a, b Container[T]) {
	for item := range a.Items() {
		if !b.Contains(item) {
			dst.Add(item)
		}
	}
}

// Computes the symmetric difference of two containers and stores the result in the destination container.
func symmetricDifference[T any](dst, a, b Container[T]) {
	for item := range a.Items() {
		if !b.Contains(item) {
			dst.Add(item)
		}
	}
	for item := range b.Items() {
		if !a.Contains(item) {
			dst.Add(item)
		}
	}
}

// Determine if two containers are equal.
func equal[T any](a, b Container[T]) bool {
	sizeA, sizeB := a.Size(), b.Size()
	if sizeA == 0 && sizeB == 0 {
		return true
	}

	if sizeA != sizeB {
		return false
	}

	for item := range a.Items() {
		if !b.Contains(item) {
			return false
		}
	}
	return true
}

// Determine if a and b are disjoint (contain no common elements)
func disjoint[T any](a, b Container[T]) bool {
	var (
		smaller = a
		larger  = b
	)
	if a.Size() > b.Size() {
		smaller = b
		larger = a
	}

	for item := range smaller.Items() {
		if larger.Contains(item) {
			return false
		}
	}
	return true
}

// Determine if a is a subset of b.
func subset[T any](a, b Container[T]) bool {
	if a.Size() > b.Size() {
		return false
	}

	for item := range a.Items() {
		if !b.Contains(item) {
			return false
		}
	}
	return true
}

// Determine if a is a superset of b.
func superset[T any](a, b Container[T]) bool {
	return subset(b, a)
}

//============================================================================
// Serialization
//============================================================================

func marshalJSON[T any](c Container[T]) ([]byte, error) {
	return json.Marshal(c.Slice())
}

func unmarshalJSON[T any](data []byte, c Container[T]) (err error) {
	slice := make([]T, 0)
	if err = json.Unmarshal(data, &slice); err != nil {
		return err
	}

	c.Add(slice...)
	return nil
}

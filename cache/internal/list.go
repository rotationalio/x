package internal

import "time"

// List is a doubly linked list (does not allow reverse traversal).
type List[K comparable, V any] struct {
	root Entry[K, V]
	len  int
}

func NewList[K comparable, V any]() *List[K, V] {
	list := new(List[K, V])
	list.Clear()
	return list
}

// Entry is an LRU cache entry that implements a doubly linked list node.
type Entry[K comparable, V any] struct {
	Key     K
	Value   V
	Expires time.Time
	prev    *Entry[K, V]
	next    *Entry[K, V]
	list    *List[K, V]
}

// Next returns the next entry in the list, or nil if there is none.
func (e *Entry[K, V]) Next() *Entry[K, V] {
	if next := e.next; e.list != nil && next != &e.list.root {
		return next
	}
	return nil
}

// Prev returns the previous entry in the list, or nil if there is none.
func (e *Entry[K, V]) Prev() *Entry[K, V] {
	if prev := e.prev; e.list != nil && prev != &e.list.root {
		return prev
	}
	return nil
}

// Expired returns true if the entry has an expiration time
// and it is currently after the expiration time.
func (e *Entry[K, V]) Expired() bool {
	return !e.Expires.IsZero() && time.Now().After(e.Expires)
}

// Clear detaches all entries from the list and resets the list to its initial state.
// Can also be used to initialize an empty list to ensure its root is correctly set.
func (l *List[K, V]) Clear() *List[K, V] {
	l.root.next = &l.root
	l.root.prev = &l.root
	l.len = 0
	return l
}

// Len returns the number of entries in the list.
// The complexity is O(1).
func (l *List[K, V]) Len() int {
	return l.len
}

// Head returns the first entry in the list, or nil if the list is empty.
func (l *List[K, V]) Head() *Entry[K, V] {
	if l.len == 0 {
		return nil
	}
	return l.root.next
}

// Tail returns the last entry in the list, or nil if the list is empty.
func (l *List[K, V]) Tail() *Entry[K, V] {
	if l.len == 0 {
		return nil
	}
	return l.root.prev
}

// Insert an entry e after at, increments l.len and returns e.
func (l *List[K, V]) InsertAfter(e, at *Entry[K, V]) *Entry[K, V] {
	e.prev = at
	e.next = at.next
	e.prev.next = e
	e.next.prev = e
	e.list = l
	l.len++
	return e
}

// Remove removes e from its list, decrements l.len and returns V.
func (l *List[K, V]) Remove(e *Entry[K, V]) V {
	e.prev.next = e.next
	e.next.prev = e.prev
	e.next = nil // avoid memory leaks
	e.prev = nil // avoid memory leaks
	e.list = nil
	l.len--

	return e.Value
}

// Move the entry e next to at.
func (l *List[K, V]) Move(e, at *Entry[K, V]) {
	if e == at {
		return
	}

	e.prev.next = e.next
	e.next.prev = e.prev

	e.prev = at
	e.next = at.next
	e.prev.next = e
	e.next.prev = e
}

// Push inserts a new element e with value v at the front of list l and returns e.
func (l *List[K, V]) Push(k K, v V, expires time.Time) *Entry[K, V] {
	// Init the list if its not already initialized.
	if l.root.next == nil {
		l.Clear()
	}

	e := &Entry[K, V]{
		Key:     k,
		Value:   v,
		Expires: expires,
	}

	return l.InsertAfter(e, &l.root)
}

// Forward moves the entry e to the front of the list. If e is not an entry in l,
// the list is not modified. The element must not be nil (or a panic will occur).
func (l *List[K, V]) Forward(e *Entry[K, V]) {
	if e.list != l || l.root.next == e {
		return
	}
	l.Move(e, &l.root)
}

package country

import (
	"errors"
)

var (
	ErrNotFound    = errors.New("country code not found")
	ErrInvalidCode = errors.New("invalid country code")
)

// An optimized trie for 3 digit country codes.
type Trie struct {
	children [26]*Trie
	value    [26]string
}

func (t *Trie) Insert(code string, country string) {
	if len(code) == 0 {
		return
	}

	if len(code) == 1 {
		t.value[code[0]-'A'] = country
		return
	}

	var child *Trie
	if child = t.children[code[0]-'A']; child == nil {
		child = &Trie{}
		t.children[code[0]-'A'] = child
	}
	child.Insert(code[1:], country)
}

func (t *Trie) Find(code string) (string, bool) {
	if len(code) == 0 {
		return "", false
	}

	if len(code) == 1 {
		if t == nil {
			return "", false
		}

		v := t.value[code[0]-'A']
		return v, v != ""
	}

	node := t.children[code[0]-'A']
	if node == nil {
		return "", false
	}

	return node.Find(code[1:])
}

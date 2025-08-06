package country

import "sync"

var (
	root *Trie
	lumu sync.Once
)

func createLookupTrie() {
	lumu.Do(func() {
		root = &Trie{}
		for _, row := range alpha2Lookup {
			for _, country := range row {
				if country != nil {
					root.Insert(country.Alpha2, country)
					root.Insert(country.Alpha3, country)
					root.Insert(country.ShortName, country)
					root.Insert(country.LongName, country)
					for _, name := range country.UnofficialNames {
						root.Insert(name, country)
					}
				}
			}
		}
	})
}

// An optimized trie for 3 digit country codes.
type Trie struct {
	children map[byte]*Trie
	value    *Country
}

func (t *Trie) Insert(name string, country *Country) {
	if len(name) == 0 {
		t.value = country
		return
	}

	if t.children == nil {
		t.children = make(map[byte]*Trie)
	}

	var child *Trie
	if child = t.children[name[0]]; child == nil {
		child = &Trie{}
		t.children[name[0]] = child
	}
	child.Insert(name[1:], country)
}

func (t *Trie) Find(name string) (*Country, bool) {
	if len(name) == 0 {
		if t == nil {
			return nil, false
		}
		return t.value, t.value != nil
	}

	if t.children == nil {
		return nil, false
	}

	node := t.children[name[0]]
	if node == nil {
		return nil, false
	}

	return node.Find(name[1:])
}

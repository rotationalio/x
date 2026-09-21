package password

import (
	"bufio"
	"embed"
	"fmt"
	"io/fs"
	"strings"
	"sync"
)

//go:embed dictionary.txt
var dictionaryFS embed.FS

var (
	// TODO: use a trie for faster lookups
	dictionary     map[string]struct{}
	dictionaryOnce sync.Once
)

// Returns true if the word is in the dictionary regardless of case.
func IsDictionaryWord(word string) bool {
	LoadDictionary()
	_, ok := dictionary[strings.ToLower(word)]
	return ok
}

func Dictionary() map[string]struct{} {
	LoadDictionary()
	return dictionary
}

func LoadDictionary() {
	dictionaryOnce.Do(func() {
		if err := loadDictionary(); err != nil {
			panic(err)
		}
	})
}

func loadDictionary() (err error) {
	dictionary = make(map[string]struct{})

	var f fs.File
	if f, err = dictionaryFS.Open("dictionary.txt"); err != nil {
		return fmt.Errorf("failed to open dictionary file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		dictionary[strings.ToLower(scanner.Text())] = struct{}{}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read dictionary file: %w", err)
	}

	return nil
}

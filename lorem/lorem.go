/*
Package lorem generates lorem ipsum placeholder text. It exposes an Ipsum struct that
draws from a fixed vocabulary of lorem ipsum words to produce random words, sentences,
paragraphs, and full documents. The amount of text generated is controlled by tuning
parameters on the struct (e.g. MinWords, MaxWords); any zero-valued parameter is filled
with a sensible default on first use.

A package-level default Ipsum seeded from the current time backs the package-level
Words, Sentences, Paragraphs, and Document functions for quick, configuration-free use.

Note that the random source backing an Ipsum is not safe for concurrent use; create a
separate Ipsum per goroutine if generating text in parallel.
*/
package lorem

import (
	"math/rand"
	"strings"
	"sync"
	"time"
)

// Default tuning parameters applied to any zero-valued field on first use.
const (
	defaultMinWords = 4
	defaultMaxWords = 12
	defaultMinSents = 3
	defaultMaxSents = 8
	defaultMinParas = 2
	defaultMaxParas = 5
)

// words is the fixed vocabulary that all generated text is drawn from.
var words = [...]string{
	"lorem", "ipsum", "dolor", "sit", "amet", "consectetur", "adipiscing",
	"elit", "sed", "do", "eiusmod", "tempor", "incididunt", "ut", "labore",
	"et", "dolore", "magna", "aliqua", "enim", "ad", "minim", "veniam",
	"quis", "nostrud", "exercitation", "ullamco", "laboris", "nisi",
	"aliquip", "ex", "ea", "commodo", "consequat", "duis", "aute", "irure",
	"in", "reprehenderit", "voluptate", "velit", "esse", "cillum", "eu",
	"fugiat", "nulla", "pariatur", "excepteur", "sint", "occaecat",
	"cupidatat", "non", "proident", "sunt", "culpa", "qui", "officia",
	"deserunt", "mollit", "anim", "id", "est", "laborum", "perspiciatis",
	"unde", "omnis", "iste", "natus", "error", "voluptatem", "accusantium",
	"doloremque", "laudantium", "totam", "rem", "aperiam", "eaque", "ipsa",
	"quae", "ab", "illo", "inventore", "veritatis", "quasi", "architecto",
	"beatae", "vitae", "dicta", "explicabo", "nemo", "ipsam", "quia",
	"voluptas", "aspernatur", "aut", "odit", "fugit", "consequuntur",
	"magni", "dolores", "eos", "ratione", "sequi", "nesciunt", "neque",
	"porro", "quisquam", "dolorem", "adipisci", "numquam", "eius", "modi",
	"tempora", "incidunt", "magnam", "aliquam", "quaerat", "minima",
	"nostrum", "exercitationem", "ullam", "corporis", "suscipit",
	"laboriosam", "aliquid", "commodi", "consequatur", "autem", "vel",
	"eum", "iure", "quam", "nihil", "molestiae", "illum", "quo", "at",
	"vero", "accusamus", "iusto", "odio", "dignissimos", "ducimus",
	"blanditiis", "praesentium", "voluptatum", "deleniti", "atque",
	"corrupti", "quos", "quas", "molestias", "excepturi", "occaecati",
	"cupiditate", "provident", "similique", "mollitia", "animi", "dolorum",
	"fuga", "harum", "quidem", "rerum", "facilis", "expedita", "distinctio",
	"nam", "libero", "tempore", "cum", "soluta", "nobis", "eligendi",
	"optio", "cumque", "impedit", "minus", "quod", "maxime", "placeat",
	"facere", "possimus", "assumenda", "repellendus", "temporibus",
	"quibusdam", "officiis", "debitis", "necessitatibus", "saepe",
	"eveniet", "voluptates", "repudiandae", "recusandae", "itaque",
	"earum", "hic", "tenetur", "a", "sapiente", "delectus", "reiciendis",
	"voluptatibus", "maiores", "alias", "perferendis", "doloribus",
	"asperiores", "repellat",
}

// Ipsum generates lorem ipsum text. The tuning parameters control how much text each
// method produces; any parameter left as its zero value is replaced with a package
// default on the first call to any method. Methods take no arguments: adjust the
// exported fields (or use New) to change behavior.
//
// The zero value is usable: an &Ipsum{} will default all parameters and seed its random
// source from the current time on first use.
type Ipsum struct {
	MinWords int // Minimum number of words in a sentence (and in a Words result).
	MaxWords int // Maximum number of words in a sentence (and in a Words result).
	MinSents int // Minimum number of sentences in a paragraph (and in a Sentences result).
	MaxSents int // Maximum number of sentences in a paragraph (and in a Sentences result).
	MinParas int // Minimum number of paragraphs in a document (and in a Paragraphs result).
	MaxParas int // Maximum number of paragraphs in a document (and in a Paragraphs result).

	rand *rand.Rand
	once sync.Once
}

// New returns an Ipsum whose random source is seeded with the given seed. Using a fixed
// seed makes the generated text deterministic, which is useful for tests. Tuning
// parameters are applied lazily on first use.
func New(seed int64) *Ipsum {
	return &Ipsum{rand: rand.New(rand.NewSource(seed))}
}

// init lazily fills any zero-valued parameter with its default and ensures the random
// source is initialized. It is invoked exactly once via sync.Once at the top of every
// method so that both New-constructed and zero-value Ipsum instances behave correctly.
func (i *Ipsum) init() {
	if i.MinWords == 0 {
		i.MinWords = defaultMinWords
	}
	if i.MaxWords == 0 {
		i.MaxWords = defaultMaxWords
	}
	if i.MinSents == 0 {
		i.MinSents = defaultMinSents
	}
	if i.MaxSents == 0 {
		i.MaxSents = defaultMaxSents
	}
	if i.MinParas == 0 {
		i.MinParas = defaultMinParas
	}
	if i.MaxParas == 0 {
		i.MaxParas = defaultMaxParas
	}
	if i.rand == nil {
		i.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
}

// Words returns a random number of words, between MinWords and MaxWords inclusive, each
// drawn at random from the fixed vocabulary.
func (i *Ipsum) Words() []string {
	i.once.Do(i.init)

	n := i.intn(i.MinWords, i.MaxWords)
	out := make([]string, n)
	for j := range out {
		out[j] = i.word()
	}
	return out
}

// Sentences returns a random number of sentences, between MinSents and MaxSents
// inclusive. Each sentence contains a random number of words between MinWords and
// MaxWords inclusive, begins with a capital letter, and ends with weighted random
// punctuation (85% ".", 10% "?", 5% "!").
func (i *Ipsum) Sentences() []string {
	i.once.Do(i.init)

	n := i.intn(i.MinSents, i.MaxSents)
	out := make([]string, n)
	for j := range out {
		out[j] = i.sentence()
	}
	return out
}

// Paragraphs returns a random number of paragraphs, between MinParas and MaxParas
// inclusive. Each paragraph is a single string containing a random number of sentences
// between MinSents and MaxSents inclusive, joined by spaces.
func (i *Ipsum) Paragraphs() []string {
	i.once.Do(i.init)

	n := i.intn(i.MinParas, i.MaxParas)
	out := make([]string, n)
	for j := range out {
		out[j] = i.paragraph()
	}
	return out
}

// Document returns a full document: a random number of paragraphs joined by blank lines.
func (i *Ipsum) Document() string {
	i.once.Do(i.init)
	return strings.Join(i.Paragraphs(), "\n\n")
}

// paragraph builds a single paragraph of MinSents..MaxSents sentences joined by spaces.
func (i *Ipsum) paragraph() string {
	n := i.intn(i.MinSents, i.MaxSents)
	sents := make([]string, n)
	for j := range sents {
		sents[j] = i.sentence()
	}
	return strings.Join(sents, " ")
}

// sentence builds a single sentence of MinWords..MaxWords words, capitalizing the first
// letter and appending weighted random punctuation.
func (i *Ipsum) sentence() string {
	n := i.intn(i.MinWords, i.MaxWords)
	parts := make([]string, n)
	for j := range parts {
		parts[j] = i.word()
	}

	sentence := strings.Join(parts, " ")
	sentence = strings.ToUpper(sentence[:1]) + sentence[1:]
	return sentence + i.punctuation()
}

// word returns a single random word from the fixed vocabulary.
func (i *Ipsum) word() string {
	return words[i.rand.Intn(len(words))]
}

// punctuation returns terminal sentence punctuation with a weighted distribution: 85%
// ".", 10% "?", and 5% "!".
func (i *Ipsum) punctuation() string {
	switch n := i.rand.Intn(100); {
	case n < 85:
		return "."
	case n < 95:
		return "?"
	default:
		return "!"
	}
}

// intn returns a random integer in [min, max] inclusive. If max is less than min, min
// is returned.
func (i *Ipsum) intn(min, max int) int {
	if max <= min {
		return min
	}
	return i.rand.Intn(max-min+1) + min
}

// ipsum is the package-level default Ipsum, seeded from the current time, that backs the
// package-level generator functions.
var ipsum = New(time.Now().UnixNano())

// Words returns random words using the default Ipsum. See Ipsum.Words.
func Words() []string { return ipsum.Words() }

// Sentences returns random sentences using the default Ipsum. See Ipsum.Sentences.
func Sentences() []string { return ipsum.Sentences() }

// Paragraphs returns random paragraphs using the default Ipsum. See Ipsum.Paragraphs.
func Paragraphs() []string { return ipsum.Paragraphs() }

// Document returns a random document using the default Ipsum. See Ipsum.Document.
func Document() string { return ipsum.Document() }

# Lorem Ipsum Text Generator

Package `lorem` generates lorem ipsum placeholder text -- random words, sentences,
paragraphs, and full documents drawn from a fixed lorem ipsum vocabulary. It has no
third-party dependencies and uses the standard library `math/rand` package.

## Install

```
go get go.rtnl.ai/x/lorem
```

## Quickstart

The package-level functions use a default generator seeded from the current time, so
you can produce text without any setup:

```go
package main

import (
	"fmt"
	"strings"

	"go.rtnl.ai/x/lorem"
)

func main() {
	fmt.Println(strings.Join(lorem.Words(), " ")) // a handful of random words
	fmt.Println(lorem.Sentences())                // []string of random sentences
	fmt.Println(lorem.Paragraphs())               // []string, one entry per paragraph
	fmt.Println(lorem.Document())                 // full document as a single string
}
```

## Configuring an `Ipsum`

The amount of text generated is controlled by tuning parameters that live directly on
the `Ipsum` struct. Methods take no arguments; set the fields you care about and any
field left as its zero value is filled with a sensible default on first use:

```go
ipsum := &lorem.Ipsum{
	MinWords: 3,  // fewest words per sentence (and per Words result)
	MaxWords: 10, // most words per sentence (and per Words result)
	MinSents: 2,  // fewest sentences per paragraph (and per Sentences result)
	MaxSents: 5,  // most sentences per paragraph (and per Sentences result)
	MinParas: 1,  // fewest paragraphs per document (and per Paragraphs result)
	MaxParas: 3,  // most paragraphs per document (and per Paragraphs result)
}

fmt.Println(ipsum.Document())
```

The zero value works too -- `&lorem.Ipsum{}` applies all defaults and seeds its random
source from the current time on first use.

### Deterministic output

For reproducible output (e.g. in tests), construct an `Ipsum` with a fixed seed using
`New`. Two generators created with the same seed produce identical text:

```go
a := lorem.New(42)
b := lorem.New(42)
// a.Document() == b.Document()
```

## Methods

| Method            | Returns    | Description                                                                 |
| ----------------- | ---------- | --------------------------------------------------------------------------- |
| `Words()`         | `[]string` | Random words, between `MinWords` and `MaxWords` inclusive.                   |
| `Sentences()`     | `[]string` | Random sentences, between `MinSents` and `MaxSents` inclusive.              |
| `Paragraphs()`    | `[]string` | Random paragraphs (one string each), between `MinParas` and `MaxParas`.     |
| `Document()`      | `string`   | A random number of paragraphs joined by blank lines.                        |

Each sentence contains a random number of words (`MinWords`..`MaxWords`), begins with a
capital letter, and ends with weighted random punctuation: `.` 85% of the time, `?` 10%,
and `!` 5%.

## Concurrency

The random source backing an `Ipsum` is not safe for concurrent use. If you need to
generate text from multiple goroutines, give each goroutine its own `Ipsum`.

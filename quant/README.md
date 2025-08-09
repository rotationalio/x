# quant

Metrics for quantitative text analysis.

GitHub: <https://go.rtnl.ai/x/quant>

## The Vision

`x/quant` will house high-performance quantitative metrics, especially ones that can be computed over text.
Think statistical or structural properties of a string or document.
The main use case is text analysis inside [Endeavor](https://github.com/rotationalio/endeavor).

`x/quant` will enable NLP for AI engineers: numeric metrics that help us reason about what a model did, what the humans expected, and how to compare the two.

We want this package to be:

* Performant (written in Go, reused across tools)
* Composable (individual functions that do one thing well)
* Extensible (easy to add new metrics as we learn more)

## Design Goals

* Each metric should be self-contained and independently callable
* Avoid hard dependencies on LLMs or external services
* Define a common interface (e.g., all metrics take `string`, return `float64` or `map[string]float64`)
* Organize by category (similarity, counts, readability, lexical, etc.)
* Stub out room for future metrics, even weird ones

## Features, metrics, and tools

* Tokenization, stemming, and type counting (see [`tokens.go`](./tokens.go))
  * Porter2/Snowball stemming algorithm
  * Regex tokenization with custom expressions
* Cosine similarity (see [`similarity.go`](./similarity.go))
* Vectors & vectorization (see [`vectors.go`](./vectors.go))
  * One-hot encoding
  * Frequency (count) encoding

### Planned

* Readability Scores (ASAP)
* Part-of-Speech Distributions (Future)
* Named Entities & Keyphrase Counts (Future)
* Custom Classifiers (Distant Future)

## API Structure

There are 3 levels in the API:

1) Level one ('low-level API') are the functions which take pre-processed data, such as a list of tokens or a vector, and perform operations on those objects, such as stemming the tokens or calculating the cosine of the angles between two vectors, with the return types being basic Go types. Example: `Cosine(a, b []float64) (cosine float64, err error)`
2) Level two ('high-level API') are the functions which take chunks of text and compose several low-level API functions to perform some operations, such as tokenizing and stemming the text to return the vocabulary or type count of the text chunk, with the return types being basic Go types. Example: `Similarity(chunkA, chunkB string, opts ...SimilarityOption) (similarity float64, err error)`
3) (Planned; not yet implemented) Level three ('document API') is a document-based API to access the high-level and low-level APIs for the same chunk of text in a single object which you only have to initialize once, and which use sub-types for those operations that usually provide a document API for their type as well. Examples: `NewDocument(chunk string, opts ...DocumentOption) (doc Document, err error)`, `Document.Tokenize() (tokens []Token, err error)`, and `Token.Stem() (stem string)`

## Developing in x/quant

Different feature categories are separated into different files, for example we might have similarity metrics in `similarity.go` and text classifiers in `classifiers.go`.
If you want to add a new feature, please ensure it is placed in a file which fits the category, or create a new file if none yet exist, and ensure the file is in `package quant`.
Tests should be located next to each feature, for example `similarity_tests.go` would hold the tests for `similarity.go`.
Tests should all be in `package quant_test` and any test data should go into the `testdata/` folder.
Documentation should go into each function's and package's docstrings so the documentation is accessible to the user while using the library in their local IDE and also available using Go's documentation tools.
Any documentation or research that isn't immediately relevant to the user in the code context should go into the `docs/` folder.

## Sources and References

To ensure the algorithms in this package are accurate, we pulled information from several references, which have been recorded in [`docs/sources.md`](./docs/sources.md) and in the documentation and comments for the individual functions in this library.

## Research Notes

Research on different topics will go into the folder [`docs/research/`](./docs/research/).

* [Go NLP](./docs/research/go_nlp.md): notes on different NLP packages/libraries for Go

## License

See: [LICENSE](../LICENSE)

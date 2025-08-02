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

### Implemented

* This package is only a stub for now.

### Planned

* Cosine Similarity (ASAP)
* Readability Scores (ASAP)
* Token & Type Counts (ASAP)
* Part-of-Speech Distributions (Future)
* Named Entities & Keyphrase Counts (Future)
* Custom Classifiers (Distant Future)

## Developing in x/quant

Different feature categories are separated into different files, for example we might have similarity metrics in `similarity.go` and text classifiers in `classifiers.go`.
If you want to add a new feature, please ensure it is placed in a file which fits the category, or create a new file if none yet exist, and ensure the file is in `package quant`.
Tests should be located next to each feature, for example `similarity_tests.go` would hold the tests for `similarity.go`.
Tests should all be in `package quant_test` and any test data should go into the `testdata/` folder.
Documentation should go into each function's and package's docstrings so the documentation is accessible to the user while using the library in their local IDE and also available using Go's documentation tools.
Any documentation or research that isn't immediately relevant to the user in the code context should go into the `docs/` folder.

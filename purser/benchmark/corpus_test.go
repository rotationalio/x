package benchmark_test

// Plaintext corpus for benchmarks: embed lines and rotate into a reused scratch buffer.

import (
	_ "embed"
	"strings"
)

//go:embed plaintext.txt
var plaintextCorpus []byte

// corpusLines holds non-empty lines from plaintext.txt, populated in init.
var corpusLines []string

// init parses the embedded plaintext corpus into corpusLines.
func init() {
	for line := range strings.SplitSeq(string(plaintextCorpus), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			corpusLines = append(corpusLines, line)
		}
	}
	if len(corpusLines) == 0 {
		corpusLines = []string{"fallback-bench-plaintext-line"}
	}
}

// nextPlain copies a rotated corpus line into scratch[:size] and returns that slice.
// scratch must have len >= size. iter selects the line with a prime stride mix.
func nextPlain(scratch []byte, size, iter int) []byte {
	if size == 0 {
		return scratch[:0]
	}
	if len(scratch) < size {
		panic("benchmark: scratch shorter than size")
	}

	n := len(corpusLines)
	idx := (iter*1103515245 + iter*4093) % n
	if idx < 0 {
		idx += n
	}
	line := corpusLines[idx]

	var off int
	for off < size {
		m := copy(scratch[off:size], line)
		if m == 0 {
			break
		}
		off += m
	}
	return scratch[:size]
}

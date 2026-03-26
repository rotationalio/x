package rlog

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"
)

// capturingJSONOpts is reused by [CapturingTestHandler.Handle] so every log line
// does not rebuild [MergeWithCustomLevels](nil) (closure + [slog.HandlerOptions] escape).
var capturingJSONOpts = MergeWithCustomLevels(nil)

// captureState holds lines and records shared by all derived handlers from the same root.
type captureState struct {
	mu      sync.Mutex
	records []slog.Record
	lines   []string
}

// CapturingTestHandler is a [slog.Handler] that records each [slog.Record] and
// each rendered JSON line (one line per Handle call). All derived handlers
// (via WithAttrs/WithGroup) share the same capture buffers. Construct with
// [NewCapturingTestHandler]; pass a non-nil [testing.TB] to also log each line
// via [testing.TB.Log].
type CapturingTestHandler struct {
	state    *captureState
	topAttrs []slog.Attr
	segments []captureGroupSegment
	tb       testing.TB
}

// captureGroupSegment is one WithGroup plus attrs added before the next WithGroup.
type captureGroupSegment struct {
	name  string
	attrs []slog.Attr
}

// NewCapturingTestHandler returns an empty [*CapturingTestHandler] with JSON line
// output. If tb is not nil, each line is also sent to [testing.TB.Log] on the
// associated [testing.TB] (e.g. for live test output).
func NewCapturingTestHandler(tb testing.TB) *CapturingTestHandler {
	if tb != nil {
		tb.Helper()
	}
	return &CapturingTestHandler{state: &captureState{}, tb: tb}
}

// Enabled always returns true so tests see all levels.
func (h *CapturingTestHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

// Handle renders one JSON line (with top attrs and groups), then appends a copy of r and
// that line to shared state under a single critical section. Nothing is stored if JSON
// rendering fails, so [CapturingTestHandler.Records] and [CapturingTestHandler.Lines] stay
// aligned by index. Concurrent Handle calls may reorder entries relative to start time,
// but each index still pairs one record with one line.
func (h *CapturingTestHandler) Handle(ctx context.Context, r slog.Record) error {
	rec := r.Clone()

	var buf bytes.Buffer
	var jh slog.Handler = slog.NewJSONHandler(&buf, capturingJSONOpts)

	if len(h.topAttrs) > 0 {
		jh = jh.WithAttrs(h.topAttrs)
	}

	for _, seg := range h.segments {
		jh = jh.WithGroup(seg.name)
		if len(seg.attrs) > 0 {
			jh = jh.WithAttrs(seg.attrs)
		}
	}

	if err := jh.Handle(ctx, r); err != nil {
		return err
	}

	line := strings.TrimSuffix(buf.String(), "\n")

	h.state.mu.Lock()
	h.state.records = append(h.state.records, rec)
	h.state.lines = append(h.state.lines, line)
	h.state.mu.Unlock()

	if h.tb != nil {
		h.tb.Log(line)
	}

	return nil
}

// WithAttrs adds attrs at the top level until a group is opened; then they attach to the innermost group.
func (h *CapturingTestHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	// If no groups are open, add the attrs to the top level.
	if len(h.segments) == 0 {
		top := h.topAttrs
		if len(top) > 0 {
			top = append(slices.Clone(top), attrs...)
		} else {
			top = attrs
		}
		return &CapturingTestHandler{state: h.state, topAttrs: top, segments: h.segments, tb: h.tb}
	}

	// If a group is open, add the attrs to the innermost group.
	segs := slices.Clone(h.segments)
	last := len(segs) - 1
	if len(segs[last].attrs) > 0 {
		segs[last].attrs = append(slices.Clone(segs[last].attrs), attrs...)
	} else {
		segs[last].attrs = attrs
	}
	return &CapturingTestHandler{state: h.state, topAttrs: h.topAttrs, segments: segs, tb: h.tb}
}

// WithGroup opens a new nested group for subsequent WithAttrs.
func (h *CapturingTestHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	segs := append(slices.Clone(h.segments), captureGroupSegment{name: name})
	return &CapturingTestHandler{state: h.state, topAttrs: h.topAttrs, segments: segs, tb: h.tb}
}

// Records returns a snapshot of captured records (newest appended last).
func (h *CapturingTestHandler) Records() []slog.Record {
	h.state.mu.Lock()
	defer h.state.mu.Unlock()
	out := make([]slog.Record, len(h.state.records))
	copy(out, h.state.records)
	return out
}

// Lines returns a snapshot of captured JSON lines (one per Handle).
func (h *CapturingTestHandler) Lines() []string {
	h.state.mu.Lock()
	defer h.state.mu.Unlock()
	out := make([]string, len(h.state.lines))
	copy(out, h.state.lines)
	return out
}

// RecordsAndLines returns snapshots of records and lines taken under one lock. Prefer this
// over separate [CapturingTestHandler.Records] and [CapturingTestHandler.Lines] calls when
// logs may be written concurrently, so lengths match and index i is always the same event.
func (h *CapturingTestHandler) RecordsAndLines() (records []slog.Record, lines []string) {
	h.state.mu.Lock()
	defer h.state.mu.Unlock()
	records = make([]slog.Record, len(h.state.records))
	copy(records, h.state.records)
	lines = make([]string, len(h.state.lines))
	copy(lines, h.state.lines)
	return records, lines
}

// Reset clears captured records and lines.
func (h *CapturingTestHandler) Reset() {
	h.state.mu.Lock()
	defer h.state.mu.Unlock()
	h.state.records = h.state.records[:0]
	h.state.lines = h.state.lines[:0]
}

// ResultMaps parses each captured line as JSON into a map.
// This is useful for use with [testing/slogtest.TestHandler].
func (h *CapturingTestHandler) ResultMaps() ([]map[string]any, error) {
	lines := h.Lines()
	out := make([]map[string]any, 0, len(lines))

	for _, line := range lines {
		m, err := ParseJSONLine(line)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}

	return out, nil
}

// ParseJSONLine unmarshals one JSON log line into a map.
func ParseJSONLine(s string) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, err
	}
	return m, nil
}

// MustParseJSONLine is like [ParseJSONLine] but panics on error; intended for tests.
func MustParseJSONLine(s string) map[string]any {
	m, err := ParseJSONLine(s)
	if err != nil {
		panic(err)
	}
	return m
}

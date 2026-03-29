// Package console implements a [slog.Handler] that pretty-prints log lines with optional colors
// and a trailing JSON object of attributes. For development and tests, not high-throughput logging.
//
// The command `go run ./rlog/console/cmd/` gives a demo of the handler with various options.
package console

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"slices"
	"sync"

	"go.rtnl.ai/x/rlog"
)

const timeFormat = "[15:04:05.000]"

// Handler writes a text prefix ([basename:line] when AddSource, bracketed time, level, message) and optional JSON attributes.
type Handler struct {
	w        io.Writer
	mu       *sync.Mutex
	opts     Options
	topAttrs []slog.Attr
	segments []segment
}

// segment is one [slog.Handler.WithGroup] name plus attributes.
type segment struct {
	name  string
	attrs []slog.Attr
}

// New creates a [Handler] writing to w. If opts is nil, all defaults are used.
func New(w io.Writer, opts *Options) *Handler {
	if opts == nil {
		opts = &Options{}
	}

	return &Handler{
		w:    w,
		opts: *opts,
		mu:   &sync.Mutex{},
	}
}

// Ensure that Handler implements the slog.Handler interface.
var _ slog.Handler = (*Handler)(nil)

// Enabled reports whether the handler is enabled for the given level.
func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	if h.opts.HandlerOptions == nil || h.opts.Level == nil {
		return level >= slog.LevelInfo
	}
	return level >= h.opts.Level.Level()
}

// WithAttrs returns a new handler with the given attributes.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	if len(h.segments) == 0 {
		top := h.topAttrs
		if len(top) > 0 {
			top = append(slices.Clone(top), attrs...)
		} else {
			top = attrs
		}
		return &Handler{w: h.w, mu: h.mu, opts: h.opts, topAttrs: top, segments: h.segments}
	}
	segs := slices.Clone(h.segments)
	last := len(segs) - 1
	if len(segs[last].attrs) > 0 {
		segs[last].attrs = append(slices.Clone(segs[last].attrs), attrs...)
	} else {
		segs[last].attrs = attrs
	}
	return &Handler{w: h.w, mu: h.mu, opts: h.opts, topAttrs: h.topAttrs, segments: segs}
}

// WithGroup returns a new handler with the given group.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	segs := append(slices.Clone(h.segments), segment{name: name})
	return &Handler{w: h.w, mu: h.mu, opts: h.opts, topAttrs: h.topAttrs, segments: segs}
}

// Handle writes the record to the handler's writer.
func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	var buf bytes.Buffer

	// ReplaceAttr (slog.HandlerOptions) can rename, drop, or rewrite built-in keys before we render.
	// Built-ins use a nil group path, matching slog's convention for top-level keys.
	var repl func([]string, slog.Attr) slog.Attr
	if h.opts.HandlerOptions != nil {
		repl = h.opts.ReplaceAttr
	}

	// Time: normalize to UTC when requested, build a TimeKey attr, then run ReplaceAttr so output
	// can omit or alter the timestamp without changing r.Time itself.
	tm := r.Time
	if h.opts.UTCTime {
		tm = tm.UTC()
	}
	var timeAttr slog.Attr
	if !tm.IsZero() {
		timeAttr = slog.Time(slog.TimeKey, tm)
		if repl != nil {
			timeAttr = repl(nil, timeAttr)
		}
	}

	// Level: wrap the record level as LevelKey and pass through ReplaceAttr (e.g. custom labels).
	levelAttr := slog.Any(slog.LevelKey, r.Level)
	if repl != nil {
		levelAttr = repl(nil, levelAttr)
	}

	// Message: wrap r.Message as MessageKey and pass through ReplaceAttr.
	msgAttr := slog.String(slog.MessageKey, r.Message)
	if repl != nil {
		msgAttr = repl(nil, msgAttr)
	}

	// Source: when AddSource is on and the record has a PC, build SourceKey and allow ReplaceAttr
	// to strip it (empty attr). If still present, emit [basename:line] before the rest of the prefix.
	showSource := false
	var srcAttr slog.Attr
	if h.opts.HandlerOptions != nil && h.opts.AddSource && r.PC != 0 {
		if src := r.Source(); src != nil {
			srcAttr = slog.Any(slog.SourceKey, src)
			if repl != nil {
				srcAttr = repl(nil, srcAttr)
			}
			showSource = !srcAttr.Equal(slog.Attr{})
		}
	}
	if showSource {
		writeSourcePrefix(&buf, h.opts.NoColor, srcAttr)
	}

	// Prefix: bracketed time (light gray) when ReplaceAttr left a non-zero time value.
	if !timeAttr.Equal(slog.Attr{}) && timeAttr.Value.Kind() == slog.KindTime && !timeAttr.Value.Time().IsZero() {
		tm2 := timeAttr.Value.Time()
		if h.opts.UTCTime {
			tm2 = tm2.UTC()
		}
		writePlainOrColor(&buf, h.opts.NoColor, lightGray, tm2.Format(timeFormat))
		buf.WriteRune(' ')
	}

	// Level prefix: when ReplaceAttr rewrites [slog.LevelKey] to a string (e.g. TRACE via
	// [rlog.MergeWithCustomLevels]), print that text plus ":" so the line matches other
	// handlers; otherwise use [slog.Level.String] from levelForOutput. ANSI color still
	// uses the resolved [slog.Level] (lv), not the string label.
	lv := levelForOutput(levelAttr, r.Level)
	var level string
	if levelAttr.Key == slog.LevelKey && !levelAttr.Equal(slog.Attr{}) && levelAttr.Value.Kind() == slog.KindString {
		if s := levelAttr.Value.String(); s != "" {
			level = s + ":"
		} else {
			level = lv.String() + ":"
		}
	} else {
		level = lv.String() + ":"
	}
	if code, ok := levelANSICode(lv); h.opts.NoColor || !ok {
		fmt.Fprint(&buf, level)
	} else {
		colorize(&buf, code, level)
	}

	buf.WriteRune(' ')

	// Message: use ReplaceAttr's string if still valid; otherwise fall back to r.Message.
	msg := r.Message
	if !msgAttr.Equal(slog.Attr{}) && msgAttr.Value.Kind() == slog.KindString {
		msg = msgAttr.Value.String()
	}
	writePlainOrColor(&buf, h.opts.NoColor, white, msg)

	// User attributes: everything on the record except the built-in keys (already shown in the prefix),
	// merged with handler-scoped attrs from WithAttrs / WithGroup.
	var recAttrs []slog.Attr
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == slog.LevelKey || a.Key == slog.MessageKey || a.Key == slog.TimeKey {
			return true
		}
		recAttrs = append(recAttrs, a)
		return true
	})
	merged := buildMergedUserAttrs(h.topAttrs, h.segments, recAttrs)

	// Trailer: one JSON object (Encode adds a trailing newline) or plain newline when NoJSON.
	if !h.opts.NoJSON {
		buf.WriteRune(' ')
		attrs := make(map[string]any)
		for _, a := range merged {
			mergeAttrJSON(attrs, nil, a, repl)
		}
		enc := json.NewEncoder(&buf)
		if h.opts.IndentJSON {
			enc.SetIndent("", "  ")
		}
		if err := enc.Encode(attrs); err != nil {
			return err
		}
	} else {
		buf.WriteRune('\n')
	}

	// Single Write call per log line; mu coordinates output when multiple handlers share one writer.
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(buf.Bytes())
	return err
}

// writePlainOrColor prints text with or without ANSI color.
func writePlainOrColor(buf *bytes.Buffer, noColor bool, color uint8, text string) {
	if noColor {
		fmt.Fprint(buf, text)
	} else {
		colorize(buf, color, text)
	}
}

// writeSourcePrefix prints [basename:line] when srcAttr carries *slog.Source (same bracket style as time).
func writeSourcePrefix(buf *bytes.Buffer, noColor bool, srcAttr slog.Attr) {
	if srcAttr.Value.Kind() != slog.KindAny {
		return
	}
	s, ok := srcAttr.Value.Any().(*slog.Source)
	if !ok || s == nil || (s.File == "" && s.Line == 0) {
		return
	}
	writePlainOrColor(buf, noColor, lightGray, fmt.Sprintf("[%s:%d]", filepath.Base(s.File), s.Line))
	buf.WriteRune(' ')
}

// levelForOutput prefers a slog.Level inside levelAttr after ReplaceAttr.
func levelForOutput(levelAttr slog.Attr, fallback slog.Level) slog.Level {
	if levelAttr.Equal(slog.Attr{}) {
		return fallback
	}
	if levelAttr.Value.Kind() == slog.KindAny {
		if l, ok := levelAttr.Value.Any().(slog.Level); ok {
			return l
		}
	}
	return fallback
}

// levelANSICode maps standard and rlog levels to a color; ok false means uncolored.
func levelANSICode(lv slog.Level) (code uint8, ok bool) {
	switch lv {
	case rlog.LevelTrace:
		return lightGray, true
	case slog.LevelDebug:
		return cyan, true
	case slog.LevelInfo:
		return lightGreen, true
	case slog.LevelWarn:
		return lightYellow, true
	case slog.LevelError:
		return lightRed, true
	case rlog.LevelFatal:
		return red, true
	case rlog.LevelPanic:
		return red, true
	default:
		return 0, false
	}
}

// buildMergedUserAttrs wraps record attrs in innermost→outermost groups, then appends top attrs.
func buildMergedUserAttrs(top []slog.Attr, segs []segment, record []slog.Attr) []slog.Attr {
	inner := slices.Clone(record)
	for i := len(segs) - 1; i >= 0; i-- {
		s := segs[i]
		args := make([]any, 0, len(s.attrs)+len(inner))
		for _, a := range s.attrs {
			args = append(args, a)
		}
		for _, a := range inner {
			args = append(args, a)
		}
		inner = []slog.Attr{slog.Group(s.name, args...)}
	}
	return append(slices.Clone(top), inner...)
}

// mergeAttrJSON flattens attrs into a JSON-ready map; ReplaceAttr gets the group path for leaves.
func mergeAttrJSON(attrs map[string]any, groups []string, a slog.Attr, repl func([]string, slog.Attr) slog.Attr) {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return
	}
	switch a.Value.Kind() {
	// Recurse into group members; extend groups for named groups.
	case slog.KindGroup:
		gattrs := a.Value.Group()
		if len(gattrs) == 0 {
			return
		}
		if a.Key != "" {
			g := append(slices.Clone(groups), a.Key)
			sub := ensureSubmap(attrs, a.Key)
			for _, ga := range gattrs {
				mergeAttrJSON(sub, g, ga, repl)
			}
		} else {
			for _, ga := range gattrs {
				mergeAttrJSON(attrs, groups, ga, repl)
			}
		}
	default:
		// Leaf: ReplaceAttr, then store or re-enter if ReplaceAttr produced a group.
		leaf := a
		if repl != nil {
			leaf = repl(groups, a)
		}
		if leaf.Equal(slog.Attr{}) {
			return
		}
		leaf.Value = leaf.Value.Resolve()
		if leaf.Value.Kind() == slog.KindGroup {
			mergeAttrJSON(attrs, groups, leaf, nil)
			return
		}
		attrs[leaf.Key] = slogValueToJSON(leaf.Value)
	}
}

// ensureSubmap returns attrs[key] as a nested map, creating it if needed.
func ensureSubmap(attrs map[string]any, key string) map[string]any {
	if existing, ok := attrs[key].(map[string]any); ok {
		return existing
	}
	sub := make(map[string]any)
	attrs[key] = sub
	return sub
}

// slogValueToJSON converts a resolved [slog.Value] for encoding/json.
func slogValueToJSON(v slog.Value) any {
	switch v.Kind() {
	case slog.KindString:
		return v.String()
	case slog.KindInt64:
		return v.Int64()
	case slog.KindUint64:
		return v.Uint64()
	case slog.KindFloat64:
		return v.Float64()
	case slog.KindBool:
		return v.Bool()
	case slog.KindTime:
		return v.Time()
	case slog.KindDuration:
		return v.Duration()
	case slog.KindAny:
		return v.Any()
	default:
		return v.String()
	}
}

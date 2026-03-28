// Package console provides a console handler for the rlog package that pretty prints
// colored text logs to stdout. It is a wrapper around the slog.Handler interface.
//
// NOTE: this should only be used for testing and development.
package console

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"go.rtnl.ai/x/rlog"
)

const (
	timeFormat = "[15:04:05.000]"
)

type Handler struct {
	slog.Handler
	w    io.Writer
	mu   *sync.Mutex
	opts Options
}

func New(w io.Writer, opts *Options) *Handler {
	if opts == nil {
		opts = &Options{}
	}

	return &Handler{
		Handler: slog.NewTextHandler(w, opts.HandlerOptions),
		w:       w,
		opts:    *opts,
		mu:      &sync.Mutex{},
	}
}

var _ slog.Handler = (*Handler)(nil)

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		Handler: h.Handler.WithAttrs(attrs),
		w:       h.w,
		mu:      h.mu, // mutex shared by all clones of this handler
		opts:    h.opts,
	}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		Handler: h.Handler.WithGroup(name),
		w:       h.w,
		mu:      h.mu, // mutex shared by all clones of this handler
		opts:    h.opts,
	}
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	// Write to a buffer first to avoid race conditions with the writer.
	// NOTE: slog uses an internal pool-allocated bytes buffer for performance.
	// Because we're allocating a new buffer here, this handler should not be used in
	// performance intensive applications.
	var buf bytes.Buffer

	// Print the timestamp.
	tm := r.Time
	if h.opts.UTCTime {
		tm = tm.UTC()
	}
	if h.opts.NoColor {
		fmt.Fprint(&buf, tm.Format(timeFormat))
	} else {
		colorize(&buf, lightGray, tm.Format(timeFormat))
	}

	// Space separator between timestamp and level.
	buf.WriteRune(' ')

	// Print the level.
	level := r.Level.String() + ":"
	if h.opts.NoColor {
		fmt.Fprint(&buf, level)
	} else {
		switch r.Level {
		case rlog.LevelTrace:
			colorize(&buf, lightGray, level)
		case slog.LevelDebug:
			colorize(&buf, cyan, level)
		case slog.LevelInfo:
			colorize(&buf, lightGreen, level)
		case slog.LevelWarn:
			colorize(&buf, lightYellow, level)
		case slog.LevelError:
			colorize(&buf, lightRed, level)
		case rlog.LevelFatal:
			colorize(&buf, red, level)
		case rlog.LevelPanic:
			colorize(&buf, red, level)
		default:
			fmt.Fprint(&buf, level)
		}
	}

	// Space separator between level and message.
	buf.WriteRune(' ')

	// Print the message.
	if h.opts.NoColor {
		fmt.Fprint(&buf, r.Message)
	} else {
		colorize(&buf, white, r.Message)
	}

	if !h.opts.NoJSON {
		// Space separator between message and JSON dictionary.
		buf.WriteRune(' ')

		// Gather the attributes.
		attrs := make(map[string]any)
		r.Attrs(func(a slog.Attr) bool {
			// Skip the level, message, and time keys.
			if a.Key == slog.LevelKey || a.Key == slog.MessageKey || a.Key == slog.TimeKey {
				return true
			}
			addAttr(attrs, a)
			return true
		})

		// Print the JSON dictionary of attributes.
		encoder := json.NewEncoder(&buf)
		if h.opts.IndentJSON {
			encoder.SetIndent("", "  ")
		}
		if err := encoder.Encode(attrs); err != nil {
			return err
		}
	} else {
		buf.WriteRune('\n')
	}

	// Lock this mutex including all handler clones so that we can write to the writer safely.
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, err := h.w.Write(buf.Bytes()); err != nil {
		return err
	}
	return nil
}

// addAttr adds an attribute to the map, capturing groups properly.
func addAttr(attrs map[string]any, a slog.Attr) {
	// Resolve the value of the attribute.
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return
	}

	// Add the attribute to the map.
	switch a.Value.Kind() {
	case slog.KindGroup:
		// Group values carry nested attrs (e.g. slog.Group("db", ...)).
		gattrs := a.Value.Group()
		if len(gattrs) == 0 {
			return
		}
		if a.Key != "" {
			// Named group: nest under this key as a JSON object, merging with any
			// prior attrs already stored for the same key.
			var sub map[string]any
			if existing, ok := attrs[a.Key].(map[string]any); ok {
				sub = existing
			} else {
				sub = make(map[string]any)
				attrs[a.Key] = sub
			}

			for _, ga := range gattrs {
				addAttr(sub, ga)
			}
		} else {
			// Anonymous / inline group: add members at the current map level (no extra nesting).
			for _, ga := range gattrs {
				addAttr(attrs, ga)
			}
		}
	default:
		attrs[a.Key] = slogValueToJSON(a.Value)
	}
}

// slogValueToJSON returns a value suitable for encoding/json from a resolved slog.Value.
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

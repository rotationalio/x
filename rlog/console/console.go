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
	if h.opts.NoColor {
		fmt.Fprint(&buf, r.Time.Format(timeFormat))
	} else {
		colorize(&buf, lightGray, r.Time.Format(timeFormat))
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
			attrs[a.Key] = a.Value.Any()
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

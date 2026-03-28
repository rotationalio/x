package fanout

import (
	"context"
	"errors"
	"log/slog"
)

// The fanout handler is a [slog.Handler] that forwards each record to every
// child handler, using a fresh [slog.Record.Clone] per child. For Go 1.26 or
// later, use the standard library's [slog.MultiHandler] instead; this is a clone
// of the functionality for Go 1.25 and earlier.
type Handler struct {
	handlers []slog.Handler
}

// New returns a new fanout [Handler] that forwards each record to every child
// handler. For Go 1.26 or later, use [slog.MultiHandler] instead; this is a
// clone of the functionality for Go 1.25 and earlier.
func New(handlers ...slog.Handler) *Handler {
	hs := append([]slog.Handler(nil), handlers...)
	return &Handler{handlers: hs}
}

// Enabled reports whether any child handler accepts the given level.
func (f *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range f.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle forwards a clone of r to each child that [slog.Handler.Enabled] accepts
// for r's level, and joins any non-nil errors.
func (f *Handler) Handle(ctx context.Context, r slog.Record) error {
	if len(f.handlers) == 0 {
		return nil
	}

	level := r.Level
	var errs []error
	for _, h := range f.handlers {
		if h.Enabled(ctx, level) {
			r2 := r.Clone()
			if err := h.Handle(ctx, r2); err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errors.Join(errs...)
}

// WithAttrs returns a [Handler] whose children are wrapped with the same attrs.
func (f *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return f
	}

	n := len(f.handlers)
	next := make([]slog.Handler, n)
	for i, h := range f.handlers {
		next[i] = h.WithAttrs(attrs)
	}

	return &Handler{handlers: next}
}

// WithGroup returns a [Handler] whose children are wrapped with the same group.
func (f *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return f
	}

	n := len(f.handlers)
	next := make([]slog.Handler, n)
	for i, h := range f.handlers {
		next[i] = h.WithGroup(name)
	}

	return &Handler{handlers: next}
}

package rlog

import (
	"context"
	"errors"
	"log/slog"
)

// FanOut is a [slog.Handler] that forwards each record to every child handler,
// using a fresh [slog.Record.Clone] per child.
type FanOut struct {
	handlers []slog.Handler
}

// NewFanOut returns a new [FanOut] that forwards each record to every child
// handler.
func NewFanOut(handlers ...slog.Handler) *FanOut {
	hs := append([]slog.Handler(nil), handlers...)
	return &FanOut{handlers: hs}
}

// Enabled reports whether any child handler accepts the given level.
func (f *FanOut) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range f.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle forwards a clone of r to each child and joins any non-nil errors.
func (f *FanOut) Handle(ctx context.Context, r slog.Record) error {
	if len(f.handlers) == 0 {
		return nil
	}

	var errs []error
	for _, h := range f.handlers {
		r2 := r.Clone()
		if err := h.Handle(ctx, r2); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// WithAttrs returns a fan-out whose children are wrapped with the same attrs.
func (f *FanOut) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return f
	}

	n := len(f.handlers)
	next := make([]slog.Handler, n)
	for i, h := range f.handlers {
		next[i] = h.WithAttrs(attrs)
	}

	return &FanOut{handlers: next}
}

// WithGroup returns a fan-out whose children are wrapped with the same group.
func (f *FanOut) WithGroup(name string) slog.Handler {
	if name == "" {
		return f
	}

	n := len(f.handlers)
	next := make([]slog.Handler, n)
	for i, h := range f.handlers {
		next[i] = h.WithGroup(name)
	}

	return &FanOut{handlers: next}
}

// Package rlog provides a wrapper around the standard library's slog package
// that adds custom severity levels and a custom exit function.
//
// Example:
//
//	opts := &slog.HandlerOptions{
//		Level: LevelTrace,
//		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
//			if a.Key == "drop" {
//				return slog.Attr{}
//			}
//			return a
//		},
//	}
//	logger := rlog.New(slog.New(slog.NewJSONHandler(w, rlog.MergeWithCustomLevels(opts))))
//	logger.Trace("message", "hello", "world", "drop", "dropped")
//	// Output example: {"level":"TRACE","msg":"message","hello":"world"}
//	logger.Fatal("message", "hello", "world", "drop", "dropped")
//	// Output example: {"level":"FATAL","msg":"message","hello":"world"} (and then calls the exit function)
//	logger.Panic("message", "hello", "world", "drop", "dropped")
//	// Output example: {"level":"PANIC","msg":"message","hello":"world"} (and then panics with "message")
package rlog

import (
	"context"
	"log/slog"
	"os"
)

// Logger is a wrapper around the standard library's slog.Logger that adds custom
// severity levels with a custom exit function for logging Fatal messages.
type Logger struct {
	*slog.Logger
	exitFunc func()
}

// New returns a new [Logger] using the given [slog.Logger].
func New(logger *slog.Logger) *Logger {
	return &Logger{Logger: logger}
}

//=============================================================================
// Trace Functions
//=============================================================================

// Trace logs at [LevelTrace]. Arguments are handled like [slog.Logger.Log]; you
// can pass any number of key/value pairs or [slog.Attr] objects.
func (l *Logger) Trace(msg string, args ...any) {
	l.TraceContext(context.Background(), msg, args...)
}

// TraceContext logs at [LevelTrace] with ctx. Arguments are handled like
// [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr]
// objects.
func (l *Logger) TraceContext(ctx context.Context, msg string, args ...any) {
	l.Logger.Log(ctx, LevelTrace, msg, args...)
}

// TraceAttrs logs at [LevelTrace] with ctx and attrs. Uses [slog.LogAttrs]
// under the hood for zero-alloc performance.
func (l *Logger) TraceAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.Logger.LogAttrs(ctx, LevelTrace, msg, attrs...)
}

//=============================================================================
// Fatal Functions
//=============================================================================

// Fatal logs at [LevelFatal] then calls the function set with [Logger.SetExitFunc]
// with the exit code set with [Logger.SetExitCode]. It does not return. Arguments
// are handled like [slog.Logger.Log]; you can pass any number of key/value
// pairs or [slog.Attr] objects.
func (l *Logger) Fatal(msg string, args ...any) {
	l.FatalContext(context.Background(), msg, args...)
}

// FatalContext logs at [LevelFatal] with ctx then calls the function set with
// [Logger.SetExitFunc] with the exit code set with [Logger.SetExitCode]. It
// does not return. Arguments  are handled like [slog.Logger.Log]; you can pass
// any number of key/value  pairs or [slog.Attr] objects.
func (l *Logger) FatalContext(ctx context.Context, msg string, args ...any) {
	l.Logger.Log(ctx, LevelFatal, msg, args...)
	l.exit()
}

// FatalAttrs logs at [LevelFatal] with attrs then calls the function set with
// [Logger.SetExitFunc] with the exit code set with [Logger.SetExitCode]. It
// does not return. Uses [slog.LogAttrs] under the hood for zero-alloc
// performance.
func (l *Logger) FatalAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.Logger.LogAttrs(ctx, LevelFatal, msg, attrs...)
	l.exit()
}

//=============================================================================
// Panic Functions
//=============================================================================

// Panic logs at [LevelPanic] then panics with msg. Arguments are handled like
// [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr]
// objects.
func (l *Logger) Panic(msg string, args ...any) {
	l.PanicContext(context.Background(), msg, args...)
}

// PanicContext logs at [LevelPanic] with ctx then panics with msg. Arguments
// are handled like [slog.Logger.Log]; you can pass any number of key/value
// pairs or [slog.Attr] objects.
func (l *Logger) PanicContext(ctx context.Context, msg string, args ...any) {
	l.Logger.Log(ctx, LevelPanic, msg, args...)
	panic(msg)
}

// PanicAttrs logs at [LevelPanic] with attrs then panics with msg. Uses
// [slog.LogAttrs] under the hood for zero-alloc performance.
func (l *Logger) PanicAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.Logger.LogAttrs(ctx, LevelPanic, msg, attrs...)
	panic(msg)
}

//=============================================================================
// Exit Function
//=============================================================================

var defaultExitFunc func() = func() { os.Exit(1) }

// SetExitFunc sets the function called by Fatal logging functions after
// logging. Useful for testing.
func (l *Logger) SetExitFunc(fn func()) {
	l.exitFunc = fn
}

// exit calls the exit function with the exit code. If the exit code is 0, it
// defaults to 1. If the exit function is nil, it defaults to [os.Exit].
func (l *Logger) exit() {
	if l.exitFunc == nil {
		defaultExitFunc()
	}
	l.exitFunc()
}

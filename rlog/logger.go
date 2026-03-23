package rlog

import (
	"context"
	"log/slog"
	"os"
	"sync"
)

//=============================================================================
// Logger Type
//=============================================================================

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
// Global Logger
//=============================================================================

// Global variables for the default (global) [Logger].
var (
	globalLogger *Logger
	loggerMu     sync.Mutex

	globalLevel *slog.LevelVar // [slog.LevelVar] is concurrency safe
)

// Default returns the default (global) [Logger], creating it if it doesn't
// exist. Use [SetDefault] to set the default [Logger]. If no default [Logger]
// is set, a new JSON logger to stdout is created with the default
// [slog.HandlerOptions] and level [slog.LevelInfo]. Safe to use concurrently.
func Default() *Logger {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	if globalLogger == nil {
		// A [slog.LevelVar] allows us to change the global log level at anytime
		globalLevel.Set(slog.LevelInfo)
		globalLogger = New(slog.New(slog.NewJSONHandler(os.Stdout, MergeWithCustomLevels(&slog.HandlerOptions{Level: globalLevel}))))
	}
	return globalLogger
}

// SetDefault sets the default (global) [Logger]. Safe to use concurrently.
func SetDefault(logger *Logger) {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	globalLogger = logger
}

// SetLevel sets the global log level. Safe to use concurrently.
func SetLevel(level slog.Level) {
	globalLevel.Set(level)
}

// Level returns the global log level. Safe to use concurrently.
func Level() slog.Level {
	return globalLevel.Level()
}

// LevelString returns the global log level as a string using a [LevelDecoder].
// Safe to use concurrently.
func LevelString() string {
	return LevelDecoder(Level()).String()
}

//=============================================================================
// Attrs Functions for default slog levels
//=============================================================================

// DebugAttrs logs at slog.LevelDebug with ctx and attrs. Uses slog.LogAttrs for efficiency.
func (l *Logger) DebugAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.Logger.LogAttrs(ctx, slog.LevelDebug, msg, attrs...)
}

// InfoAttrs logs at slog.LevelInfo with ctx and attrs. Uses slog.LogAttrs for efficiency.
func (l *Logger) InfoAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.Logger.LogAttrs(ctx, slog.LevelInfo, msg, attrs...)
}

// WarnAttrs logs at slog.LevelWarn with ctx and attrs. Uses slog.LogAttrs for efficiency.
func (l *Logger) WarnAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.Logger.LogAttrs(ctx, slog.LevelWarn, msg, attrs...)
}

// ErrorAttrs logs at slog.LevelError with ctx and attrs. Uses slog.LogAttrs for efficiency.
func (l *Logger) ErrorAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.Logger.LogAttrs(ctx, slog.LevelError, msg, attrs...)
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

// TraceAttrs logs at [LevelTrace] with ctx and attrs. Uses [slog.LogAttrs] for
// efficiency.
func (l *Logger) TraceAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.Logger.LogAttrs(ctx, LevelTrace, msg, attrs...)
}

//=============================================================================
// Fatal Functions
//=============================================================================

// Fatal logs at [LevelFatal] then calls the exit hook if set, otherwise [os.Exit]
// with code 1. It does not return. Arguments are handled like [slog.Logger.Log];
// you can pass any number of key/value pairs or [slog.Attr] objects.
func (l *Logger) Fatal(msg string, args ...any) {
	l.FatalContext(context.Background(), msg, args...)
}

// FatalContext logs at [LevelFatal] with ctx then calls the exit hook if set,
// otherwise [os.Exit] with code 1. It does not return. Arguments are handled like
// [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr]
// objects.
func (l *Logger) FatalContext(ctx context.Context, msg string, args ...any) {
	l.Logger.Log(ctx, LevelFatal, msg, args...)
	l.exit()
}

// FatalAttrs logs at [LevelFatal] with attrs then calls the exit hook if set,
// otherwise [os.Exit] with code 1. It does not return. Uses [slog.LogAttrs] for
// efficiency.
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
// [slog.LogAttrs] for efficiency.
func (l *Logger) PanicAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.Logger.LogAttrs(ctx, LevelPanic, msg, attrs...)
	panic(msg)
}

//=============================================================================
// Exit Function
//=============================================================================

var defaultExitFunc func() = func() { os.Exit(1) }

// SetExitFunc sets the function called after Fatal logging. If unset, Fatal
// calls [os.Exit] with code 1. Useful for testing.
func (l *Logger) SetExitFunc(fn func()) {
	l.exitFunc = fn
}

// exit runs the configured exit hook or [os.Exit] with code 1.
func (l *Logger) exit() {
	if l.exitFunc == nil {
		defaultExitFunc()
		return // unreachable after os.Exit; satisfies nil-check analysis
	}
	l.exitFunc()
}

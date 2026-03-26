package rlog

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
)

//=============================================================================
// Logger Type
//=============================================================================

// Logger is a wrapper around the standard library's [slog.Logger] that adds custom
// severity levels (Trace, Fatal, Panic). After Fatal logging, see [SetFatalHook].
// The wrapped [slog.Logger] is stored in the [Logger] struct and can be accessed
// using the [Logger.Logger] field, though it is not recommended to mutate the
// wrapped [slog.Logger] directly.
type Logger struct {
	*slog.Logger
}

// New returns a new [Logger] wrapping the given [slog.Logger]. If logger is nil,
// returns the [Default] [Logger].
func New(logger *slog.Logger) *Logger {
	if logger == nil {
		return Default()
	}
	return &Logger{Logger: logger}
}

// With returns a derived [Logger] with the given context attributes (key/value
// pairs and/or [slog.Attr] values, same rules as [slog.Logger.With]).
func (l *Logger) With(args ...any) *Logger {
	return &Logger{Logger: l.Logger.With(args...)}
}

// WithGroup returns a derived [Logger] with the given group.
func (l *Logger) WithGroup(name string) *Logger {
	return &Logger{Logger: l.Logger.WithGroup(name)}
}

//=============================================================================
// Global Logger Init and Management
//=============================================================================

var (
	// Stores the global logger that is returned by [Default]
	globalLogger *Logger
	// Protects reads and writes to the globalLogger
	loggerMu sync.RWMutex
	// The zero value is [slog.LevelInfo]
	globalLevel *slog.LevelVar = &slog.LevelVar{}
	// Stores the function to call when a fatal log is written, or nil if no hook is set
	fatalHookAtomic atomic.Pointer[struct{ fn func() }]
)

// Initializes the global logger to be a console JSON logger with level [slog.LevelInfo].
func init() {
	SetDefault(New(slog.New(slog.NewJSONHandler(os.Stdout, MergeWithCustomLevels(&slog.HandlerOptions{Level: globalLevel})))))
}

// Default returns the default (global) [Logger]. Use [SetDefault] to set the
// default [Logger]. The returned pointer is shared until the next [SetDefault];
// do not mutate the [Logger] value it points to.
func Default() *Logger {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	return globalLogger
}

// SetDefault sets the default (global) [Logger] and [slog.Default] to the same
// underlying [*slog.Logger]. To use the global logger level and custom levels,
// the handler must be constructed using [MergeWithCustomLevels] and
// [WithGlobalLevel]. Safe to use concurrently.
func SetDefault(logger *Logger) {
	p := new(Logger)
	*p = *logger
	loggerMu.Lock()
	defer loggerMu.Unlock()
	globalLogger = p
	slog.SetDefault(p.Logger)
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

// SetLevel sets the global log level. Safe to use concurrently.
func SetLevel(level slog.Level) {
	globalLevel.Set(level)
}

// SetFatalHook sets the function invoked after Fatal log output. If fn is nil,
// the hook is cleared and Fatal calls [os.Exit](1). The hook is process-wide
// for all [*Logger] values. Safe to call concurrently with logging.
func SetFatalHook(fn func()) {
	if fn == nil {
		fatalHookAtomic.Store(nil)
		return
	}
	fatalHookAtomic.Store(&struct{ fn func() }{fn: fn})
}

// exitFatal runs the hook from [SetFatalHook] if installed, else [os.Exit](1).
func exitFatal() {
	slot := fatalHookAtomic.Load()
	if slot == nil {
		os.Exit(1)
	}
	slot.fn()
}

//=============================================================================
// Global Logger Functions
// NOTE: These functions are aliases for the [Logger] methods with the default [Logger].
//=============================================================================

// With returns a derived [Logger] with the given attributes.
func With(args ...any) *Logger {
	return Default().With(args...)
}

// WithGroup returns a derived [Logger] with the given group.
func WithGroup(name string) *Logger {
	return Default().WithGroup(name)
}

// Log emits a log record with the given level and message using the [Default] logger.
// Arguments are handled like [slog.Logger.Log].
func Log(ctx context.Context, level slog.Level, msg string, args ...any) {
	Default().Log(ctx, level, msg, args...)
}

// LogAttrs is like [Log] but accepts attrs only; it uses [slog.LogAttrs] for efficiency.
func LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	Default().LogAttrs(ctx, level, msg, attrs...)
}

// Trace logs at [LevelTrace] using the [Default] logger. Arguments are handled like
// [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr] objects.
func Trace(msg string, args ...any) {
	Default().Trace(msg, args...)
}

// Debug logs at [slog.LevelDebug] using the [Default] logger. Arguments are handled like
// [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr] objects.
func Debug(msg string, args ...any) {
	Default().Debug(msg, args...)
}

// Info logs at [slog.LevelInfo] using the [Default] logger. Arguments are handled like
// [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr] objects.
func Info(msg string, args ...any) {
	Default().Info(msg, args...)
}

// Warn logs at [slog.LevelWarn] using the [Default] logger. Arguments are handled like
// [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr] objects.
func Warn(msg string, args ...any) {
	Default().Warn(msg, args...)
}

// Error logs at [slog.LevelError] using the [Default] logger. Arguments are handled like
// [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr] objects.
func Error(msg string, args ...any) {
	Default().Error(msg, args...)
}

// Fatal logs at [LevelFatal] using the [Default] logger, then runs the global fatal hook
// from [SetFatalHook] if set, otherwise [os.Exit](1). It does not return. Arguments are
// handled like [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr] objects.
func Fatal(msg string, args ...any) {
	Default().Fatal(msg, args...)
}

// Panic logs at [LevelPanic] using the [Default] logger, then panics with msg. Arguments are
// handled like [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr]
// objects.
func Panic(msg string, args ...any) {
	Default().Panic(msg, args...)
}

// TraceContext logs at [LevelTrace] with ctx using the [Default] logger. Arguments are handled
// like [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr] objects.
func TraceContext(ctx context.Context, msg string, args ...any) {
	Default().TraceContext(ctx, msg, args...)
}

// DebugContext logs at [slog.LevelDebug] with ctx using the [Default] logger. Arguments are
// handled like [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr]
// objects.
func DebugContext(ctx context.Context, msg string, args ...any) {
	Default().DebugContext(ctx, msg, args...)
}

// InfoContext logs at [slog.LevelInfo] with ctx using the [Default] logger. Arguments are
// handled like [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr]
// objects.
func InfoContext(ctx context.Context, msg string, args ...any) {
	Default().InfoContext(ctx, msg, args...)
}

// WarnContext logs at [slog.LevelWarn] with ctx using the [Default] logger. Arguments are
// handled like [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr]
// objects.
func WarnContext(ctx context.Context, msg string, args ...any) {
	Default().WarnContext(ctx, msg, args...)
}

// ErrorContext logs at [slog.LevelError] with ctx using the [Default] logger. Arguments are
// handled like [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr]
// objects.
func ErrorContext(ctx context.Context, msg string, args ...any) {
	Default().ErrorContext(ctx, msg, args...)
}

// FatalContext logs at [LevelFatal] with ctx using the [Default] logger, then runs the global
// fatal hook from [SetFatalHook] if set, otherwise [os.Exit](1). It does not return. Arguments
// are handled like [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr] objects.
func FatalContext(ctx context.Context, msg string, args ...any) {
	Default().FatalContext(ctx, msg, args...)
}

// PanicContext logs at [LevelPanic] with ctx using the [Default] logger, then panics with msg.
// Arguments are handled like [slog.Logger.Log]; you can pass any number of key/value pairs or
// [slog.Attr] objects.
func PanicContext(ctx context.Context, msg string, args ...any) {
	Default().PanicContext(ctx, msg, args...)
}

// TraceAttrs logs at [LevelTrace] with ctx and attrs using the [Default] logger. Uses
// [slog.LogAttrs] for efficiency.
func TraceAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	Default().TraceAttrs(ctx, msg, attrs...)
}

// DebugAttrs logs at [slog.LevelDebug] with ctx and attrs using the [Default] logger. Uses
// [slog.LogAttrs] for efficiency.
func DebugAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	Default().DebugAttrs(ctx, msg, attrs...)
}

// InfoAttrs logs at [slog.LevelInfo] with ctx and attrs using the [Default] logger. Uses
// [slog.LogAttrs] for efficiency.
func InfoAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	Default().InfoAttrs(ctx, msg, attrs...)
}

// WarnAttrs logs at [slog.LevelWarn] with ctx and attrs using the [Default] logger. Uses
// [slog.LogAttrs] for efficiency.
func WarnAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	Default().WarnAttrs(ctx, msg, attrs...)
}

// ErrorAttrs logs at [slog.LevelError] with ctx and attrs using the [Default] logger. Uses
// [slog.LogAttrs] for efficiency.
func ErrorAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	Default().ErrorAttrs(ctx, msg, attrs...)
}

// FatalAttrs logs at [LevelFatal] with attrs using the [Default] logger, then runs the global
// fatal hook from [SetFatalHook] if set, otherwise [os.Exit](1). It does not return. Uses
// [slog.LogAttrs] for efficiency.
func FatalAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	Default().FatalAttrs(ctx, msg, attrs...)
}

// PanicAttrs logs at [LevelPanic] with attrs using the [Default] logger, then panics with msg.
// Uses [slog.LogAttrs] for efficiency.
func PanicAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	Default().PanicAttrs(ctx, msg, attrs...)
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

// Fatal logs at [LevelFatal] then runs the global fatal hook from [SetFatalHook] if set,
// otherwise [os.Exit](1). It does not return. Arguments are handled like [slog.Logger.Log];
// you can pass any number of key/value pairs or [slog.Attr] objects.
func (l *Logger) Fatal(msg string, args ...any) {
	l.FatalContext(context.Background(), msg, args...)
}

// FatalContext logs at [LevelFatal] with ctx then runs the global fatal hook from [SetFatalHook]
// if set, otherwise [os.Exit](1). It does not return. Arguments are handled like
// [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr]
// objects.
func (l *Logger) FatalContext(ctx context.Context, msg string, args ...any) {
	l.Logger.Log(ctx, LevelFatal, msg, args...)
	exitFatal()
}

// FatalAttrs logs at [LevelFatal] with attrs then runs the global fatal hook from [SetFatalHook]
// if set, otherwise [os.Exit](1). It does not return. Uses [slog.LogAttrs] for
// efficiency.
func (l *Logger) FatalAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.Logger.LogAttrs(ctx, LevelFatal, msg, attrs...)
	exitFatal()
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

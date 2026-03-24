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
// Global Logger Init and Management
//=============================================================================

// Global variables for the default (global) [Logger].
var (
	globalLogger Logger
	loggerMu     sync.RWMutex   // protects reads and writes to the globalLogger
	globalLevel  *slog.LevelVar = &slog.LevelVar{}
)

// Initializes the global logger and level once. Is a no-op if already initialized.
func init() {
	globalLevel.Set(slog.LevelInfo)
	globalLogger = *newDefaultGlobalLogger()
}

// Default returns the default (global) [Logger], creating it if it doesn't
// exist. Use [SetDefault] to set the default [Logger]. If no default [Logger]
// is set, a new JSON logger to stdout is created with the default
// [slog.HandlerOptions] and level [slog.LevelInfo]. Safe to use concurrently.
func Default() *Logger {
	loggerMu.RLock()
	l := globalLogger
	loggerMu.RUnlock()
	return &l
}

// SetDefault sets the default (global) [Logger]. To use the global logger level
// and custom levels, the handler must be constructed using [MergeWithCustomLevels]
// and [WithGlobalLevel]. Safe to use concurrently.
func SetDefault(logger *Logger) {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	globalLogger = *logger
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

// newDefaultGlobalLogger creates a new default global [Logger] with a JSON stdout
// handler with the global level.
func newDefaultGlobalLogger() *Logger {
	return New(slog.New(slog.NewJSONHandler(os.Stdout, MergeWithCustomLevels(&slog.HandlerOptions{Level: globalLevel}))))
}

//=============================================================================
// Global Logger Functions
// NOTE: These functions are aliases for the [Logger] methods with the default [Logger].
//=============================================================================

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

// Fatal logs at [LevelFatal] using the [Default] logger, then calls the exit hook if set,
// otherwise [os.Exit] with code 1. It does not return. Arguments are handled like
// [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr] objects.
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

// FatalContext logs at [LevelFatal] with ctx using the [Default] logger, then calls the exit
// hook if set, otherwise [os.Exit] with code 1. It does not return. Arguments are handled like
// [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr] objects.
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

// FatalAttrs logs at [LevelFatal] with attrs using the [Default] logger, then calls the exit
// hook if set, otherwise [os.Exit] with code 1. It does not return. Uses [slog.LogAttrs] for
// efficiency.
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

// Package rlog wraps [log/slog] with extra levels (Trace, Fatal, Panic), test
// capture helpers, and small slog utilities (handler options, global level sync,
// multi-sink fan-out). See [README.md in the repo], or [rlog on pkg.go.dev], for
// API summary and usage notes.
//
//	package main
//
//	import (
//			"context"
//			"log/slog"
//			"os"
//
//			"go.rtnl.ai/x/rlog"
//	)
//
//	func main() {
//		// Global default logger (JSON stdout): allow Trace.
//		rlog.SetLevel(rlog.LevelTrace)
//		rlog.Info("hello")
//		// Output example: {"time":"2026-03-25T12:00:00.000-00:00","level":"INFO","msg":"hello"}
//		rlog.Trace("verbose")
//		// Output example: {"time":"…","level":"TRACE","msg":"verbose"}
//
//		// Custom logger: MergeWithCustomLevels names TRACE/FATAL/PANIC; WithGlobalLevel ties
//		// level to rlog.SetLevel so this handler follows the same threshold as the default.
//		opts := rlog.MergeWithCustomLevels(rlog.WithGlobalLevel(nil))
//		log := rlog.New(slog.New(slog.NewJSONHandler(os.Stdout, opts)))
//
//		// rlog-only level + key/value fields (Trace also has TraceContext, TraceAttrs, …).
//		log.Trace("trace-msg", "k", "v")
//		// Output example: {"time":"…","level":"TRACE","msg":"trace-msg","k":"v"}
//
//		// Without a hook, Fatal would os.Exit(1) and the rest of main would not run.
//		rlog.SetFatalHook(func() {}) // demo only — replace with test hook or omit in production
//		defer rlog.SetFatalHook(nil) // setting the fatal hook to nil causes future Fatal calls to use os.Exit(1)
//		log.Fatal("fatal-msg")
//		// Output example: {"time":"…","level":"FATAL","msg":"fatal-msg"} — then runs the fatal hook (no exit here)
//
//		// Panic levels call panic("panic-msg") after logging
//		func() {
//			defer func() { recover() }()
//			log.Panic("panic-msg")
//			// Output example: {"time":"…","level":"PANIC","msg":"panic-msg"} — then panic("panic-msg")
//		}()
//
//		// WithGroup/With return *rlog.Logger and work the same as for a slog.Logger.
//		sub := log.WithGroup("svc").With("name", "api")
//		sub.Info("scoped")
//		// Output example: {"time":"…","level":"INFO","msg":"scoped","svc":{"name":"api"}}
//
//		// Stdlib-style context and slog.Attr APIs on *rlog.Logger.
//		log.DebugContext(context.Background(), "debug-msg", "k", "v")
//		// Output example: {"time":"…","level":"DEBUG","msg":"debug-msg","k":"v"}
//		log.InfoAttrs(context.Background(), "info-msg", slog.String("k", "v"))
//		// Output example: {"time":"…","level":"INFO","msg":"info-msg","k":"v"}
//
//		// Package-level helpers and slog.Default now use this logger.
//		new := log.With(slog.String("new", "logger"))
//		rlog.SetDefault(new)
//		rlog.Warn("via package after SetDefault")
//		// Output example: {"time":"…","level":"WARN","msg":"via package after SetDefault","new":"logger"}
//		rlog.FatalAttrs("fatal-msg", slog.String("fatal", "attr"))
//		// Output example: {"time":"…","level":"FATAL","msg":"fatal-msg","new":"logger","fatal":"attr"} — then runs the fatal hook (no exit here)
//	}
//
// [README.md in the repo]: https://github.com/rotationalio/x/blob/main/rlog/README.md
// [rlog on pkg.go.dev]: https://pkg.go.dev/go.rtnl.ai/x/rlog
package rlog

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
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

// emit writes a record using a caller PC already captured at the public API
// boundary; behavior matches [slog.Logger]'s internal log path (nil ctx, Enabled,
// NewRecord, Handle).
func (l *Logger) emit(ctx context.Context, level slog.Level, msg string, pc uintptr, args ...any) {
	if ctx == nil {
		ctx = context.Background()
	}
	if !l.Enabled(ctx, level) {
		return
	}
	r := slog.NewRecord(time.Now(), level, msg, pc)
	r.Add(args...)
	_ = l.Handler().Handle(ctx, r)
}

// emitAttrs is like [Logger.emit] but for attribute-only records.
func (l *Logger) emitAttrs(ctx context.Context, level slog.Level, msg string, pc uintptr, attrs ...slog.Attr) {
	if ctx == nil {
		ctx = context.Background()
	}
	if !l.Enabled(ctx, level) {
		return
	}
	r := slog.NewRecord(time.Now(), level, msg, pc)
	r.AddAttrs(attrs...)
	_ = l.Handler().Handle(ctx, r)
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
// Logger logging methods
//=============================================================================

// Log emits a log record with the given level and message. Arguments are handled like
// [slog.Logger.Log].
func (l *Logger) Log(ctx context.Context, level slog.Level, msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	l.emit(ctx, level, msg, pcs[0], args...)
}

// LogAttrs is like [Logger.Log] but accepts attrs only; attribute handling matches
// [slog.Logger.LogAttrs].
func (l *Logger) LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	l.emitAttrs(ctx, level, msg, pcs[0], attrs...)
}

// Trace logs at [LevelTrace]. Arguments are handled like [slog.Logger.Log]; you can pass any
// number of key/value pairs or [slog.Attr] objects.
func (l *Logger) Trace(msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	l.emit(context.Background(), LevelTrace, msg, pcs[0], args...)
}

// TraceAttrs logs at [LevelTrace] with ctx and attrs. Attribute handling matches
// [slog.Logger.LogAttrs].
func (l *Logger) TraceAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	l.emitAttrs(ctx, LevelTrace, msg, pcs[0], attrs...)
}

// TraceContext logs at [LevelTrace] with ctx. Arguments are handled like [slog.Logger.Log]; you
// can pass any number of key/value pairs or [slog.Attr] objects.
func (l *Logger) TraceContext(ctx context.Context, msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	l.emit(ctx, LevelTrace, msg, pcs[0], args...)
}

// Debug logs at [slog.LevelDebug]. Arguments are handled like [slog.Logger.Log]; you can pass
// any number of key/value pairs or [slog.Attr] objects.
func (l *Logger) Debug(msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	l.emit(context.Background(), slog.LevelDebug, msg, pcs[0], args...)
}

// DebugAttrs logs at [slog.LevelDebug] with ctx and attrs. Attribute handling matches
// [slog.Logger.LogAttrs].
func (l *Logger) DebugAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	l.emitAttrs(ctx, slog.LevelDebug, msg, pcs[0], attrs...)
}

// DebugContext logs at [slog.LevelDebug] with ctx. Arguments are handled like [slog.Logger.Log];
// you can pass any number of key/value pairs or [slog.Attr] objects.
func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	l.emit(ctx, slog.LevelDebug, msg, pcs[0], args...)
}

// Info logs at [slog.LevelInfo]. Arguments are handled like [slog.Logger.Log]; you can pass any
// number of key/value pairs or [slog.Attr] objects.
func (l *Logger) Info(msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	l.emit(context.Background(), slog.LevelInfo, msg, pcs[0], args...)
}

// InfoAttrs logs at [slog.LevelInfo] with ctx and attrs. Attribute handling matches
// [slog.Logger.LogAttrs].
func (l *Logger) InfoAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	l.emitAttrs(ctx, slog.LevelInfo, msg, pcs[0], attrs...)
}

// InfoContext logs at [slog.LevelInfo] with ctx. Arguments are handled like [slog.Logger.Log];
// you can pass any number of key/value pairs or [slog.Attr] objects.
func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	l.emit(ctx, slog.LevelInfo, msg, pcs[0], args...)
}

// Warn logs at [slog.LevelWarn]. Arguments are handled like [slog.Logger.Log]; you can pass any
// number of key/value pairs or [slog.Attr] objects.
func (l *Logger) Warn(msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	l.emit(context.Background(), slog.LevelWarn, msg, pcs[0], args...)
}

// WarnAttrs logs at [slog.LevelWarn] with ctx and attrs. Attribute handling matches
// [slog.Logger.LogAttrs].
func (l *Logger) WarnAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	l.emitAttrs(ctx, slog.LevelWarn, msg, pcs[0], attrs...)
}

// WarnContext logs at [slog.LevelWarn] with ctx. Arguments are handled like [slog.Logger.Log];
// you can pass any number of key/value pairs or [slog.Attr] objects.
func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	l.emit(ctx, slog.LevelWarn, msg, pcs[0], args...)
}

// Error logs at [slog.LevelError]. Arguments are handled like [slog.Logger.Log]; you can pass any
// number of key/value pairs or [slog.Attr] objects.
func (l *Logger) Error(msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	l.emit(context.Background(), slog.LevelError, msg, pcs[0], args...)
}

// ErrorAttrs logs at [slog.LevelError] with ctx and attrs. Attribute handling matches
// [slog.Logger.LogAttrs].
func (l *Logger) ErrorAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	l.emitAttrs(ctx, slog.LevelError, msg, pcs[0], attrs...)
}

// ErrorContext logs at [slog.LevelError] with ctx. Arguments are handled like [slog.Logger.Log];
// you can pass any number of key/value pairs or [slog.Attr] objects.
func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	l.emit(ctx, slog.LevelError, msg, pcs[0], args...)
}

// Fatal logs at [LevelFatal], then runs the global fatal hook from [SetFatalHook] if set,
// otherwise [os.Exit](1). It does not return. Arguments are handled like [slog.Logger.Log]; you
// can pass any number of key/value pairs or [slog.Attr] objects.
func (l *Logger) Fatal(msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	l.emit(context.Background(), LevelFatal, msg, pcs[0], args...)
	exitFatal()
}

// FatalAttrs logs at [LevelFatal] with ctx and attrs, then runs the global fatal hook from
// [SetFatalHook] if set, otherwise [os.Exit](1). It does not return. Attribute handling matches
// [slog.Logger.LogAttrs].
func (l *Logger) FatalAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	l.emitAttrs(ctx, LevelFatal, msg, pcs[0], attrs...)
	exitFatal()
}

// FatalContext logs at [LevelFatal] with ctx, then runs the global fatal hook from [SetFatalHook]
// if set, otherwise [os.Exit](1). It does not return. Arguments are handled like
// [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr] objects.
func (l *Logger) FatalContext(ctx context.Context, msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	l.emit(ctx, LevelFatal, msg, pcs[0], args...)
	exitFatal()
}

// Panic logs at [LevelPanic], then panics with msg. Arguments are handled like [slog.Logger.Log];
// you can pass any number of key/value pairs or [slog.Attr] objects.
func (l *Logger) Panic(msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	l.emit(context.Background(), LevelPanic, msg, pcs[0], args...)
	panic(msg)
}

// PanicAttrs logs at [LevelPanic] with ctx and attrs, then panics with msg. Attribute handling
// matches [slog.Logger.LogAttrs].
func (l *Logger) PanicAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	l.emitAttrs(ctx, LevelPanic, msg, pcs[0], attrs...)
	panic(msg)
}

// PanicContext logs at [LevelPanic] with ctx, then panics with msg. Arguments are handled like
// [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr] objects.
func (l *Logger) PanicContext(ctx context.Context, msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	l.emit(ctx, LevelPanic, msg, pcs[0], args...)
	panic(msg)
}

//=============================================================================
// Global Logger Functions
// NOTE: These functions are aliases for the [Logger] methods with the [Default] [Logger].
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
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	Default().emit(ctx, level, msg, pcs[0], args...)
}

// LogAttrs is like [Log] but accepts attrs only; it uses [slog.LogAttrs] for efficiency.
func LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	Default().emitAttrs(ctx, level, msg, pcs[0], attrs...)
}

// Trace logs at [LevelTrace] using the [Default] logger. Arguments are handled like
// [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr] objects.
func Trace(msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	Default().emit(context.Background(), LevelTrace, msg, pcs[0], args...)
}

// TraceAttrs logs at [LevelTrace] with ctx and attrs using the [Default] logger. Uses
// [slog.LogAttrs] for efficiency.
func TraceAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	Default().emitAttrs(ctx, LevelTrace, msg, pcs[0], attrs...)
}

// TraceContext logs at [LevelTrace] with ctx using the [Default] logger. Arguments are handled
// like [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr] objects.
func TraceContext(ctx context.Context, msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	Default().emit(ctx, LevelTrace, msg, pcs[0], args...)
}

// Debug logs at [slog.LevelDebug] using the [Default] logger. Arguments are handled like
// [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr] objects.
func Debug(msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	Default().emit(context.Background(), slog.LevelDebug, msg, pcs[0], args...)
}

// DebugAttrs logs at [slog.LevelDebug] with ctx and attrs using the [Default] logger. Uses
// [slog.LogAttrs] for efficiency.
func DebugAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	Default().emitAttrs(ctx, slog.LevelDebug, msg, pcs[0], attrs...)
}

// DebugContext logs at [slog.LevelDebug] with ctx using the [Default] logger. Arguments are
// handled like [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr]
// objects.
func DebugContext(ctx context.Context, msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	Default().emit(ctx, slog.LevelDebug, msg, pcs[0], args...)
}

// Info logs at [slog.LevelInfo] using the [Default] logger. Arguments are handled like
// [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr] objects.
func Info(msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	Default().emit(context.Background(), slog.LevelInfo, msg, pcs[0], args...)
}

// InfoAttrs logs at [slog.LevelInfo] with ctx and attrs using the [Default] logger. Uses
// [slog.LogAttrs] for efficiency.
func InfoAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	Default().emitAttrs(ctx, slog.LevelInfo, msg, pcs[0], attrs...)
}

// InfoContext logs at [slog.LevelInfo] with ctx using the [Default] logger. Arguments are
// handled like [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr]
// objects.
func InfoContext(ctx context.Context, msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	Default().emit(ctx, slog.LevelInfo, msg, pcs[0], args...)
}

// Warn logs at [slog.LevelWarn] using the [Default] logger. Arguments are handled like
// [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr] objects.
func Warn(msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	Default().emit(context.Background(), slog.LevelWarn, msg, pcs[0], args...)
}

// WarnAttrs logs at [slog.LevelWarn] with ctx and attrs using the [Default] logger. Uses
// [slog.LogAttrs] for efficiency.
func WarnAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	Default().emitAttrs(ctx, slog.LevelWarn, msg, pcs[0], attrs...)
}

// WarnContext logs at [slog.LevelWarn] with ctx using the [Default] logger. Arguments are
// handled like [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr]
// objects.
func WarnContext(ctx context.Context, msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	Default().emit(ctx, slog.LevelWarn, msg, pcs[0], args...)
}

// Error logs at [slog.LevelError] using the [Default] logger. Arguments are handled like
// [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr] objects.
func Error(msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	Default().emit(context.Background(), slog.LevelError, msg, pcs[0], args...)
}

// ErrorAttrs logs at [slog.LevelError] with ctx and attrs using the [Default] logger. Uses
// [slog.LogAttrs] for efficiency.
func ErrorAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	Default().emitAttrs(ctx, slog.LevelError, msg, pcs[0], attrs...)
}

// ErrorContext logs at [slog.LevelError] with ctx using the [Default] logger. Arguments are
// handled like [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr]
// objects.
func ErrorContext(ctx context.Context, msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	Default().emit(ctx, slog.LevelError, msg, pcs[0], args...)
}

// Fatal logs at [LevelFatal] using the [Default] logger, then runs the global fatal hook
// from [SetFatalHook] if set, otherwise [os.Exit](1). It does not return. Arguments are
// handled like [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr] objects.
func Fatal(msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	Default().emit(context.Background(), LevelFatal, msg, pcs[0], args...)
	exitFatal()
}

// FatalAttrs logs at [LevelFatal] with attrs using the [Default] logger, then runs the global
// fatal hook from [SetFatalHook] if set, otherwise [os.Exit](1). It does not return. Uses
// [slog.LogAttrs] for efficiency.
func FatalAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	Default().emitAttrs(ctx, LevelFatal, msg, pcs[0], attrs...)
	exitFatal()
}

// FatalContext logs at [LevelFatal] with ctx using the [Default] logger, then runs the global
// fatal hook from [SetFatalHook] if set, otherwise [os.Exit](1). It does not return. Arguments
// are handled like [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr] objects.
func FatalContext(ctx context.Context, msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	Default().emit(ctx, LevelFatal, msg, pcs[0], args...)
	exitFatal()
}

// Panic logs at [LevelPanic] using the [Default] logger, then panics with msg. Arguments are
// handled like [slog.Logger.Log]; you can pass any number of key/value pairs or [slog.Attr]
// objects.
func Panic(msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	Default().emit(context.Background(), LevelPanic, msg, pcs[0], args...)
	panic(msg)
}

// PanicAttrs logs at [LevelPanic] with attrs using the [Default] logger, then panics with msg.
// Uses [slog.LogAttrs] for efficiency.
func PanicAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	Default().emitAttrs(ctx, LevelPanic, msg, pcs[0], attrs...)
	panic(msg)
}

// PanicContext logs at [LevelPanic] with ctx using the [Default] logger, then panics with msg.
// Arguments are handled like [slog.Logger.Log]; you can pass any number of key/value pairs or
// [slog.Attr] objects.
func PanicContext(ctx context.Context, msg string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	Default().emit(ctx, LevelPanic, msg, pcs[0], args...)
	panic(msg)
}

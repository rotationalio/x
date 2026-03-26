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

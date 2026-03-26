# rlog

The `r` is for “Rotational”! This package extends [`log/slog`](https://pkg.go.dev/log/slog) with **Trace**, **Fatal**, and **Panic**, and helpers in the usual shapes (key/value, `*Context`, `*Attrs`). See [`go doc`](https://pkg.go.dev/go.rtnl.ai/x/rlog) for the full API.

- **Fatal** runs [`SetFatalHook`](https://pkg.go.dev/go.rtnl.ai/x/rlog#SetFatalHook) after logging if set; otherwise [`os.Exit(1)`](https://pkg.go.dev/os#Exit). **Panic** logs then panics with the message.
- [`MergeWithCustomLevels`](https://pkg.go.dev/go.rtnl.ai/x/rlog#MergeWithCustomLevels) maps custom levels to names `TRACE`, `FATAL`, `PANIC` in JSON/text output. [`ReplaceLevelKey`](https://pkg.go.dev/go.rtnl.ai/x/rlog#ReplaceLevelKey) does only the level mapping if you compose `ReplaceAttr` yourself. [`WithGlobalLevel`](https://pkg.go.dev/go.rtnl.ai/x/rlog#WithGlobalLevel) helps keep the global logging level synced.
- [`Logger.With`](https://pkg.go.dev/go.rtnl.ai/x/rlog#Logger.With) / [`WithGroup`](https://pkg.go.dev/go.rtnl.ai/x/rlog#Logger.WithGroup) keep a `*rlog.Logger` (same args as [`slog.Logger.With`](https://pkg.go.dev/log/slog#Logger.With)). Package-level [`With`](https://pkg.go.dev/go.rtnl.ai/x/rlog#With), [`WithGroup`](https://pkg.go.dev/go.rtnl.ai/x/rlog#WithGroup), [`Log`](https://pkg.go.dev/go.rtnl.ai/x/rlog#Log), and [`LogAttrs`](https://pkg.go.dev/go.rtnl.ai/x/rlog#LogAttrs) use the default logger.
- [`NewFanOut`](https://pkg.go.dev/go.rtnl.ai/x/rlog#NewFanOut) returns a [`slog.Handler`](https://pkg.go.dev/log/slog#Handler) that forwards each record to every child handler (multi-sink logging).
- [`NewCapturingTestHandler`](https://pkg.go.dev/go.rtnl.ai/x/rlog#NewCapturingTestHandler) helps tests capture records and JSON lines; pass `nil` or a [`testing.TB`](https://pkg.go.dev/testing#TB) to print test logs to the console. Helpers include [`ParseJSONLine`](https://pkg.go.dev/go.rtnl.ai/x/rlog#ParseJSONLine), [`ResultMaps`](https://pkg.go.dev/go.rtnl.ai/x/rlog#CapturingTestHandler.ResultMaps), and [`RecordsAndLines`](https://pkg.go.dev/go.rtnl.ai/x/rlog#CapturingTestHandler.RecordsAndLines) which returns a mutex-locked snapshot of current logs.
- [`LevelDecoder`](https://pkg.go.dev/go.rtnl.ai/x/rlog#LevelDecoder) parses level strings, including the added `TRACE`, `FATAL`, and `PANIC`.

## Default logger and Custom Handlers

- The package default logs JSON to stdout at **Info**.
- [`SetDefault`](https://pkg.go.dev/go.rtnl.ai/x/rlog#SetDefault) replaces rlog’s global logger and updates [`slog.Default`](https://pkg.go.dev/log/slog#Default) via [`slog.SetDefault`](https://pkg.go.dev/log/slog#SetDefault).
- [`SetLevel`](https://pkg.go.dev/go.rtnl.ai/x/rlog#SetLevel), [`Level`](https://pkg.go.dev/go.rtnl.ai/x/rlog#Level), and [`LevelString`](https://pkg.go.dev/go.rtnl.ai/x/rlog#LevelString) read or set the shared [`slog.LevelVar`](https://pkg.go.dev/log/slog#LevelVar) wired into the default install.
- Handlers you build yourself should use `MergeWithCustomLevels(WithGlobalLevel(opts))` so they still honor that threshold after you call [`SetDefault`](https://pkg.go.dev/go.rtnl.ai/x/rlog#SetDefault); see [`WithGlobalLevel`](https://pkg.go.dev/go.rtnl.ai/x/rlog#WithGlobalLevel).

## Example

```go
package main

import (
 "context"
 "log/slog"
 "os"

 "go.rtnl.ai/x/rlog"
)

func main() {
 // Global default logger (JSON stdout): allow Trace.
 rlog.SetLevel(rlog.LevelTrace)
 rlog.Info("hello")
 // Output example: {"time":"2026-03-25T12:00:00.000-00:00","level":"INFO","msg":"hello"}
 rlog.Trace("verbose")
 // Output example: {"time":"…","level":"TRACE","msg":"verbose"}

 // Custom logger: MergeWithCustomLevels names TRACE/FATAL/PANIC; WithGlobalLevel ties
 // level to rlog.SetLevel so this handler follows the same threshold as the default.
 opts := rlog.MergeWithCustomLevels(rlog.WithGlobalLevel(nil))
 log := rlog.New(slog.New(slog.NewJSONHandler(os.Stdout, opts)))

 // rlog-only level + key/value fields (Trace also has TraceContext, TraceAttrs, …).
 log.Trace("trace-msg", "k", "v")
 // Output example: {"time":"…","level":"TRACE","msg":"trace-msg","k":"v"}

 // Without a hook, Fatal would os.Exit(1) and the rest of main would not run.
 rlog.SetFatalHook(func() {}) // demo only — replace with test hook or omit in production
 defer rlog.SetFatalHook(nil) // setting the fatal hook to nil causes future Fatal calls to use os.Exit(1)
 log.Fatal("fatal-msg")
 // Output example: {"time":"…","level":"FATAL","msg":"fatal-msg"} — then runs the fatal hook (no exit here)

// Panic levels call panic("panic-msg") after logging
 func() {
  defer func() { recover() }()
  log.Panic("panic-msg")
  // Output example: {"time":"…","level":"PANIC","msg":"panic-msg"} — then panic("panic-msg")
 }()

 // WithGroup/With return *rlog.Logger and work the same as for a slog.Logger.
 sub := log.WithGroup("svc").With("name", "api")
 sub.Info("scoped")
 // Output example: {"time":"…","level":"INFO","msg":"scoped","svc":{"name":"api"}}

 // Stdlib-style context and slog.Attr APIs on *rlog.Logger.
 log.DebugContext(context.Background(), "debug-msg", "k", "v")
 // Output example: {"time":"…","level":"DEBUG","msg":"debug-msg","k":"v"}
 log.InfoAttrs(context.Background(), "info-msg", slog.String("k", "v"))
 // Output example: {"time":"…","level":"INFO","msg":"info-msg","k":"v"}


 // Package-level helpers and slog.Default now use this logger.
 new := log.With(slog.String("new", "logger"))
 rlog.SetDefault(new)
 rlog.Warn("via package after SetDefault")
 // Output example: {"time":"…","level":"WARN","msg":"via package after SetDefault","new":"logger"}
 rlog.FatalAttrs("fatal-msg", slog.String("fatal", "attr"))
 // Output example: {"time":"…","level":"FATAL","msg":"fatal-msg","new":"logger","fatal":"attr"} — then runs the fatal hook (no exit here)
}
```

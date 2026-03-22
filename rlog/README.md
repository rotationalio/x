# rlog

**rlog** wraps Go’s [`log/slog`](https://pkg.go.dev/log/slog) with extra severity levels—**Trace** (more verbose than Debug), **Fatal**, and **Panic**—and helpers that mirror slog’s patterns (key/value `Log`, `LogAttrs`-style methods, and context variants). **Fatal** logs then runs an exit hook (default `os.Exit(1)`); **Panic** logs then `panic`s with the message string. Use `rlog.MergeWithCustomLevels` on your `slog.HandlerOptions` so handlers emit readable level names (`TRACE`, `FATAL`, `PANIC`) instead of numeric offsets for those levels.

## Setup

Set `HandlerOptions.Level` high enough for the lines you need (use `rlog.LevelTrace` to allow Trace). Pass `rlog.MergeWithCustomLevels(opts)` into your handler constructor. Wrap the resulting `*slog.Logger` with `rlog.New`. Optionally call `SetExitFunc` on the `*rlog.Logger` to control what happens after Fatal (default: `os.Exit(1)`).

## Example

The program below shows how to construct a logger and each **rlog-added** method, with comments giving example **JSON** output (the `time` value is from the default JSON handler and changes every run).

```go
package main

import (
 "context"
 "log/slog"
 "os"

 "go.rtnl.ai/x/rlog"
)

func main() {
 opts := &slog.HandlerOptions{
  // Minimum level: Trace is the most verbose level in this package.
  Level: rlog.LevelTrace,
 }
 // MergeWithCustomLevels: maps custom levels to level strings TRACE / FATAL / PANIC in output.
 handler := slog.NewJSONHandler(os.Stdout, rlog.MergeWithCustomLevels(opts))

 log := rlog.New(slog.New(handler))

 // Fatal* would normally exit the process; use a no-op in tests or demos.
 log.SetExitFunc(func() {})

 ctx := context.Background()

 // --- Standard slog levels via Attrs (context + slog.Attr, uses LogAttrs) ---

 log.DebugAttrs(ctx, "debug-msg", slog.String("k", "v"))
 // Output: {"time":"2006-01-02T15:04:05.000000-07:00","level":"DEBUG","msg":"debug-msg","k":"v"}

 log.InfoAttrs(ctx, "info-msg", slog.String("k", "v"))
 // Output: {"time":"...","level":"INFO","msg":"info-msg","k":"v"}

 log.WarnAttrs(ctx, "warn-msg", slog.String("k", "v"))
 // Output: {"time":"...","level":"WARN","msg":"warn-msg","k":"v"}

 log.ErrorAttrs(ctx, "error-msg", slog.String("k", "v"))
 // Output: {"time":"...","level":"ERROR","msg":"error-msg","k":"v"}

 // --- Trace (rlog.LevelTrace) ---

 log.Trace("trace-msg", "k", "v")
 // Output: {"time":"...","level":"TRACE","msg":"trace-msg","k":"v"}

 log.TraceContext(ctx, "trace-ctx", "k", "v")
 // Output: {"time":"...","level":"TRACE","msg":"trace-ctx","k":"v"}

 log.TraceAttrs(ctx, "trace-attrs", slog.String("k", "v"))
 // Output: {"time":"...","level":"TRACE","msg":"trace-attrs","k":"v"}

 // --- Fatal (logs, then SetExitFunc; default without SetExitFunc is os.Exit(1)) ---

 log.Fatal("fatal-msg", "k", "v")
 // Output: {"time":"...","level":"FATAL","msg":"fatal-msg","k":"v"}

 log.FatalContext(ctx, "fatal-ctx", "k", "v")
 // Output: {"time":"...","level":"FATAL","msg":"fatal-ctx","k":"v"}

 log.FatalAttrs(ctx, "fatal-attrs", slog.String("k", "v"))
 // Output: {"time":"...","level":"FATAL","msg":"fatal-attrs","k":"v"}

 // --- Panic (logs, then panic(message)) ---

 func() {
  defer func() { recover() }()
  log.Panic("panic-msg", "k", "v")
  // Output: {"time":"...","level":"PANIC","msg":"panic-msg","k":"v"}
  // Then: panic with value "panic-msg"
 }()

 func() {
  defer func() { recover() }()
  log.PanicContext(ctx, "panic-ctx", "k", "v")
  // Output: {"time":"...","level":"PANIC","msg":"panic-ctx","k":"v"}
  // Then: panic with value "panic-ctx"
 }()

 func() {
  defer func() { recover() }()
  log.PanicAttrs(ctx, "panic-attrs", slog.String("k", "v"))
  // Output: {"time":"...","level":"PANIC","msg":"panic-attrs","k":"v"}
  // Then: panic with value "panic-attrs"
 }()

 // The embedded *slog.Logger still provides Debug, Info, Warn, Error, Log, With, etc.
}
```

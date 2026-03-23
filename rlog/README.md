# rlog

The `r` is for “Rotational”!

## Features Overview

- Wraps [`log/slog`](https://pkg.go.dev/log/slog) with extra severities **Trace** (more verbose than Debug), **Fatal**, and **Panic**, plus helpers in the usual slog shapes: key/value, `*Context`, and `*Attrs` ([`slog.LogAttrs`](https://pkg.go.dev/log/slog#LogAttrs)).
- **Fatal** logs, then runs an exit hook (default `os.Exit(1)`); **Panic** logs, then `panic`s with the message.
- Use `rlog.MergeWithCustomLevels` on handler options so JSON/text output uses level names `TRACE`, `FATAL`, and `PANIC` instead of numeric offsets.

## Global logger

- The default global logger will log JSON to stdout at level **Info**
- `SetDefault` replaces the global logger; `Default` returns the current one. Package-level helpers (`rlog.Info`, `rlog.Trace`, `rlog.DebugContext`, …) delegate to the global logger.
- `SetLevel`, `Level`, and `LevelString` read or update the shared [`slog.LevelVar`](https://pkg.go.dev/log/slog#LevelVar) used by the default install.
- `WithGlobalLevel` sets `HandlerOptions.Level` to that same `LevelVar`. Use `MergeWithCustomLevels(WithGlobalLevel(opts))` when creating handlers so `SetLevel` still controls verbosity after `SetDefault`. A fixed `HandlerOptions{Level: …}` does not follow `SetLevel`.

## Example

Below is an example program that exercises the main features of rlog.

```go
package main

import (
 "context"
 "log/slog"
 "os"

 "go.rtnl.ai/x/rlog"
)

func main() {
 // --- Global default (before SetDefault): shared LevelVar + package-level helpers ---
 rlog.SetLevel(rlog.LevelTrace)
 rlog.Info("installed JSON default at init")
 // Output: {"time":"…","level":"INFO","msg":"installed JSON default at init"}
 rlog.Trace("package-level Trace")
 // Output: {"time":"…","level":"TRACE","msg":"package-level Trace"}

 // --- Build a logger and use *Logger methods (handler uses global LevelVar via WithGlobalLevel) ---
 opts := rlog.MergeWithCustomLevels(rlog.WithGlobalLevel(nil))
 h := slog.NewJSONHandler(os.Stdout, opts)
 log := rlog.New(slog.New(h))
 log.SetExitFunc(func() {}) // omit in production; avoids exit in this demo

 ctx := context.Background()

 // Rlog-added level: key/value (Trace, Fatal, Panic have *Context and *Attrs too)
 log.Trace("trace-msg", "k", "v")
 // Output: {"time":"…","level":"TRACE","msg":"trace-msg","k":"v"}

 // Standard slog level + context (same idea for InfoContext, WarnContext, …)
 log.DebugContext(ctx, "debug-msg", "k", "v")
 // Output: {"time":"…","level":"DEBUG","msg":"debug-msg","k":"v"}

 // Standard slog level + slog.Attr (LogAttrs-style; same for WarnAttrs, ErrorAttrs, …)
 log.InfoAttrs(ctx, "info-msg", slog.String("k", "v"))
 // Output: {"time":"…","level":"INFO","msg":"info-msg","k":"v"}

 log.Fatal("fatal-msg") // would os.Exit(1) without SetExitFunc
 // Output: {"time":"…","level":"FATAL","msg":"fatal-msg"}

 func() {
  defer func() { recover() }()
  log.Panic("panic-msg") // logs then panic(message)
  // Output: {"time":"…","level":"PANIC","msg":"panic-msg"}
 }()

 // --- Point package-level API at this logger ---
 rlog.SetDefault(log)
 rlog.Default().Warn("same logger as SetDefault")
 // Output: {"time":"…","level":"WARN","msg":"same logger as SetDefault"}
 rlog.DebugContext(ctx, "same logger via package-level DebugContext")
 // Output: {"time":"…","level":"DEBUG","msg":"same logger via package-level DebugContext"}
 rlog.InfoAttrs(ctx, "same logger via package-level InfoAttrs", slog.String("k", "v"))
 // Output: {"time":"…","level":"INFO","msg":"same logger via package-level InfoAttrs","k":"v"}
}
```

JSON output uses the default JSON handler’s `time` field; levels show as `TRACE`, `DEBUG`, `INFO`, etc.

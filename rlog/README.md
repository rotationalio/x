# rlog

The `r` is for “Rotational”! This module extends [`log/slog`](https://pkg.go.dev/log/slog) with **Trace**, **Fatal**, and **Panic**, options so those levels print with sensible names, and helpers to keep a global level in sync. The root package is [`go.rtnl.ai/x/rlog`](https://pkg.go.dev/go.rtnl.ai/x/rlog). Subpackages add a dev-friendly console handler, test capture helpers, and logging to multiple handlers at once. See [`go doc`](https://pkg.go.dev/go.rtnl.ai/x/rlog) and the subpackage docs for the full API.

## `rlog.Logger`

[`Logger`](https://pkg.go.dev/go.rtnl.ai/x/rlog#Logger) wraps [`*slog.Logger`](https://pkg.go.dev/log/slog#Logger) and adds **Trace**, **Fatal**, and **Panic** (with `Context` and `Attrs` variants), plus **DebugAttrs**, **InfoAttrs**, **WarnAttrs**, and **ErrorAttrs** for the built-in levels. [`New`](https://pkg.go.dev/go.rtnl.ai/x/rlog#New) builds one from an `*slog.Logger`; `nil` selects [`Default`](https://pkg.go.dev/go.rtnl.ai/x/rlog#Default).

[`With`](https://pkg.go.dev/go.rtnl.ai/x/rlog#Logger.With) and [`WithGroup`](https://pkg.go.dev/go.rtnl.ai/x/rlog#Logger.WithGroup) behave like slog’s. Package-level helpers delegate to the default logger.

- **Fatal** runs [`SetFatalHook`](https://pkg.go.dev/go.rtnl.ai/x/rlog#SetFatalHook) after logging if set; otherwise [`os.Exit(1)`](https://pkg.go.dev/os#Exit). **Panic** logs then panics with the message.

## Package `rlog` (core)

- [`MergeWithCustomLevels`](https://pkg.go.dev/go.rtnl.ai/x/rlog#MergeWithCustomLevels) labels custom severities as `TRACE`, `FATAL`, and `PANIC` in output. [`WithGlobalLevel`](https://pkg.go.dev/go.rtnl.ai/x/rlog#WithGlobalLevel) ties a handler’s threshold to [`SetLevel`](https://pkg.go.dev/go.rtnl.ai/x/rlog#SetLevel). [`ReplaceLevelKey`](https://pkg.go.dev/go.rtnl.ai/x/rlog#ReplaceLevelKey) is there if you build your own `ReplaceAttr` pipeline.
- [`LevelDecoder`](https://pkg.go.dev/go.rtnl.ai/x/rlog#LevelDecoder) parses level strings, including the extra severities.

## Subpackage `console`

[`go.rtnl.ai/x/rlog/console`](https://pkg.go.dev/go.rtnl.ai/x/rlog/console) provides a [`slog.Handler`](https://pkg.go.dev/log/slog#Handler) that prints **readable lines**: optional file/line, time, level (colors optional), message, and a JSON blob of attributes unless you turn that off. Meant for local dev and tests, not maximum throughput.

[`console.New`](https://pkg.go.dev/go.rtnl.ai/x/rlog/console#New) and [`console.Options`](https://pkg.go.dev/go.rtnl.ai/x/rlog/console#Options) cover the usual slog options plus color, JSON layout, and UTC time. Use [`(*Options).MergeWithCustomLevels`](https://pkg.go.dev/go.rtnl.ai/x/rlog/console#Options.MergeWithCustomLevels) so TRACE/FATAL/PANIC match the rest of rlog.

Demo from this repo: `go run ./rlog/console/cmd/`.

## Subpackage `testing`

[`go.rtnl.ai/x/rlog/testing`](https://pkg.go.dev/go.rtnl.ai/x/rlog/testing) — import with an alias (e.g. `rlogtesting "go.rtnl.ai/x/rlog/testing"`) so it does not clash with the standard [`testing`](https://pkg.go.dev/testing) package.

[`NewCapturingTestHandler`](https://pkg.go.dev/go.rtnl.ai/x/rlog/testing#NewCapturingTestHandler) captures each log as JSON; pass `t` to also print lines in the test output, or `nil` for silent capture. Use [`Records`](https://pkg.go.dev/go.rtnl.ai/x/rlog/testing#CapturingTestHandler.Records), [`Lines`](https://pkg.go.dev/go.rtnl.ai/x/rlog/testing#CapturingTestHandler.Lines), [`RecordsAndLines`](https://pkg.go.dev/go.rtnl.ai/x/rlog/testing#CapturingTestHandler.RecordsAndLines), [`ResultMaps`](https://pkg.go.dev/go.rtnl.ai/x/rlog/testing#CapturingTestHandler.ResultMaps), and [`Reset`](https://pkg.go.dev/go.rtnl.ai/x/rlog/testing#CapturingTestHandler.Reset) in assertions. [`ParseJSONLine`](https://pkg.go.dev/go.rtnl.ai/x/rlog/testing#ParseJSONLine) / [`MustParseJSONLine`](https://pkg.go.dev/go.rtnl.ai/x/rlog/testing#MustParseJSONLine) help parse lines.

Use [`rlog.New`](https://pkg.go.dev/go.rtnl.ai/x/rlog#New) on top when you need `*rlog.Logger` in tests.

## Subpackage `fanout`

[`go.rtnl.ai/x/rlog/fanout`](https://pkg.go.dev/go.rtnl.ai/x/rlog/fanout) sends **every log to every** child [`slog.Handler`](https://pkg.go.dev/log/slog#Handler). Build one with [`fanout.New`](https://pkg.go.dev/go.rtnl.ai/x/rlog/fanout#New).

On **Go 1.26+**, use [`slog.MultiHandler`](https://pkg.go.dev/log/slog#MultiHandler) instead; this package covers the same idea on **Go 1.25 and earlier**.

## Default logger and custom handlers

- Default: JSON on stdout at **Info**.
- [`SetDefault`](https://pkg.go.dev/go.rtnl.ai/x/rlog#SetDefault) replaces rlog’s default and keeps [`slog.Default`](https://pkg.go.dev/log/slog#Default) aligned.
- [`SetLevel`](https://pkg.go.dev/go.rtnl.ai/x/rlog#SetLevel), [`Level`](https://pkg.go.dev/go.rtnl.ai/x/rlog#Level), and [`LevelString`](https://pkg.go.dev/go.rtnl.ai/x/rlog#LevelString) control the default logger’s threshold.
- For your own handlers, combine `MergeWithCustomLevels` and `WithGlobalLevel` so they respect the same global level after `SetDefault` (see [`WithGlobalLevel`](https://pkg.go.dev/go.rtnl.ai/x/rlog#WithGlobalLevel)).

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
 defer rlog.SetFatalHook(nil)
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

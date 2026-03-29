// Command console demonstrates [console.Handler] and every [console.Options] field.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go.rtnl.ai/x/rlog"
	"go.rtnl.ai/x/rlog/console"
)

func main() {
	ctx := context.Background()

	banner("nil *Options (handler defaults: INFO min, color, compact JSON, local time)")
	log := slog.New(console.New(os.Stdout, nil))
	sample(ctx, log)

	banner("explicit defaults: &console.Options{}")
	log = slog.New(console.New(os.Stdout, &console.Options{}))
	sample(ctx, log)

	banner("NoColor: plain text, no ANSI")
	log = slog.New(console.New(os.Stdout, &console.Options{NoColor: true}))
	sample(ctx, log)

	banner("IndentJSON: multi-line JSON trailer")
	log = slog.New(console.New(os.Stdout, &console.Options{IndentJSON: true}))
	sample(ctx, log)

	banner("NoJSON: message only, newline (no attribute object)")
	log = slog.New(console.New(os.Stdout, &console.Options{NoJSON: true}))
	sample(ctx, log)

	banner("UTCTime: bracketed clock in UTC")
	log = slog.New(console.New(os.Stdout, &console.Options{UTCTime: true}))
	log.Info("server tick", "tz", "expect UTC in prefix")

	banner("HandlerOptions.Level: floor at TRACE (debug/trace on; FATAL/PANIC severities)")
	hOpts := &slog.HandlerOptions{Level: rlog.LevelTrace}
	log = slog.New(console.New(os.Stdout, &console.Options{HandlerOptions: hOpts}))
	log.Log(ctx, rlog.LevelTrace, "trace line")
	log.Debug("debug line")
	log.Info("info line")
	log.Log(ctx, rlog.LevelFatal, "fatal line (logged only; no exit)")
	log.Log(ctx, rlog.LevelPanic, "panic line (logged only; no panic)")

	banner("HandlerOptions.AddSource: [basename:line] before timestamp")
	log = slog.New(console.New(os.Stdout, &console.Options{
		HandlerOptions: &slog.HandlerOptions{AddSource: true},
	}))
	log.Info("called from main with AddSource")

	banner("HandlerOptions.ReplaceAttr: drop secrets, custom time/message (with MergeWithCustomLevels)")
	customReplace := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == "secret" {
			return slog.Attr{}
		}
		if len(groups) == 0 && a.Key == slog.MessageKey {
			return slog.String(slog.MessageKey, "[wrapped] "+a.Value.String())
		}
		return a
	}
	base := &slog.HandlerOptions{
		Level:       slog.LevelDebug,
		AddSource:   true,
		ReplaceAttr: customReplace,
	}
	log = slog.New(console.New(os.Stdout, &console.Options{
		HandlerOptions: rlog.MergeWithCustomLevels(base),
		NoColor:        true,
		IndentJSON:     true,
		UTCTime:        true,
	}))
	log.Info("replace attr demo", "secret", "hunter2", "public", "ok")
	log.Log(ctx, rlog.LevelTrace, "trace after MergeWithCustomLevels (level key still renders as slog level string)")

	banner("WithAttrs / WithGroup on logger")
	log = slog.New(console.New(os.Stdout, &console.Options{IndentJSON: true})).
		With("service", "demo").
		WithGroup("http").
		With(slog.String("method", "GET"))
	log.Info("grouped request", slog.String("path", "/v1/status"), "status", 200)

	banner("JSON trailer: times, durations, and nested maps in attrs")
	log = slog.New(console.New(os.Stdout, &console.Options{IndentJSON: true}))
	log.Info("mixed value kinds in JSON",
		slog.Time("when", time.Now()),
		slog.Duration("elapsed", 123*time.Millisecond),
		slog.Any("nested", map[string]int{"a": 1}),
	)
}

func banner(title string) {
	fmt.Println()
	fmt.Println("---", title, "---")
}

func sample(ctx context.Context, log *slog.Logger) {
	log.Debug("filtered at default INFO minimum")
	log.Info("hello console", "count", 3, "ok", true)
	log.Warn("something odd", slog.Duration("wait", 50*time.Millisecond))
	log.Error("failed", "err", "boom", "code", 500)
	log.Log(ctx, rlog.LevelTrace, "also filtered unless Level is lowered")
}

// Command console demonstrates [console.Handler] and [console.Options] through
// [rlog.Logger], including custom levels (TRACE, FATAL, PANIC).
package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"go.rtnl.ai/x/rlog"
	"go.rtnl.ai/x/rlog/console"
)

func main() {
	// Fatal would normally exit; keep the demo running through all sections.
	rlog.SetFatalHook(func() {})
	defer rlog.SetFatalHook(nil)

	// One pattern everywhere: banner → newConsole(opts) → standardDemo (optional extra / derive).
	run := func(title string, opts *console.Options, extra func(log *rlog.Logger)) {
		banner(title)
		log := newConsole(opts)
		standardDemo(log)
		if extra != nil {
			extra(log)
		}
	}
	runDerived := func(title string, opts *console.Options, derive func(*rlog.Logger) *rlog.Logger, extra func(log *rlog.Logger)) {
		banner(title)
		log := derive(newConsole(opts))
		standardDemo(log)
		if extra != nil {
			extra(log)
		}
	}

	run("nil *Options — defaults: INFO floor, color, compact JSON, local time", nil, nil)

	run("NoJSON — message only, no attribute object", &console.Options{NoJSON: true}, nil)

	run("HandlerOptions.Level — TRACE floor (debug/trace on)",
		&console.Options{HandlerOptions: &slog.HandlerOptions{Level: rlog.LevelTrace}}, nil)

	run("HandlerOptions.AddSource — [file:line] before timestamp",
		&console.Options{HandlerOptions: &slog.HandlerOptions{AddSource: true}}, nil)

	run("ReplaceAttr + NoColor + UTCTime — drop keys, wrap message; plain text, UTC clock",
		&console.Options{
			HandlerOptions: &slog.HandlerOptions{
				Level:       rlog.LevelTrace,
				AddSource:   true,
				ReplaceAttr: replaceAttrDemo,
			},
			NoColor: true,
			UTCTime: true,
		},
		func(log *rlog.Logger) {
			log.Info("'secret' key redacted", "secret", "NOT REDACTED!!!", "public", "visible")
			log.Trace("trace after merge still uses rlog level string (TRACE)")
		},
	)

	runDerived("IndentJSON — multi-line trailer; With / WithGroup; time, duration, maps; level Error",
		&console.Options{IndentJSON: true, HandlerOptions: &slog.HandlerOptions{Level: rlog.LevelError}},
		func(log *rlog.Logger) *rlog.Logger {
			return log.With("service", "demo").
				WithGroup("http").
				With(slog.String("method", "GET"))
		},
		func(log *rlog.Logger) {
			log.Info("request", slog.String("path", "/v1/status"), "status", 200)
			log.Info("structured attrs",
				slog.Time("when", time.Now()),
				slog.Duration("elapsed", 123*time.Millisecond),
				slog.Any("nested", map[string]int{"a": 1}),
			)
		},
	)
}

func newConsole(opts *console.Options) *rlog.Logger {
	return rlog.New(slog.New(console.New(os.Stdout, opts.MergeWithCustomLevels())))
}

func banner(title string) {
	fmt.Println()
	fmt.Println("---", title, "---")
}

func standardDemo(log *rlog.Logger) {
	log.Trace("trace")
	log.Debug("debug")
	log.Info("hello console", "count", 3, "ok", true)
	log.Warn("something odd", slog.Duration("wait", 50*time.Millisecond))
	log.Error("failed", "err", "boom", "code", 500)
	log.Fatal("fatal")
	func() {
		defer func() { recover() }()
		log.Panic("panic")
	}()
}

func replaceAttrDemo(groups []string, a slog.Attr) slog.Attr {
	if a.Key == "secret" {
		return slog.Attr{}
	}
	if len(groups) == 0 && a.Key == slog.MessageKey {
		return slog.String(slog.MessageKey, "[wrapped] "+a.Value.String())
	}
	return a
}

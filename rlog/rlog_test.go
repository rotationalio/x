package rlog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/rlog"
)

// JSON output uses "TRACE", "FATAL", and "PANIC" for the custom levels and the
// usual default slog levels plus an undefined custom level.
func TestMergeHandlerOptions_levelStrings(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(t, &buf, &slog.HandlerOptions{Level: rlog.LevelTrace})

	logger.Log(context.Background(), rlog.LevelTrace, "trace-msg")
	logger.Log(context.Background(), slog.LevelDebug, "debug-msg")
	logger.Log(context.Background(), slog.LevelInfo, "info-msg")
	logger.Log(context.Background(), slog.LevelWarn, "warn-msg")
	logger.Log(context.Background(), slog.LevelError, "error-msg")
	logger.Log(context.Background(), rlog.LevelFatal, "fatal-msg")
	logger.Log(context.Background(), rlog.LevelPanic, "panic-msg")
	logger.Log(context.Background(), rlog.LevelPanic+4, "panic-plus-4-msg")

	out := buf.String()
	for _, want := range []string{`"level":"TRACE"`, `"level":"DEBUG"`, `"level":"INFO"`, `"level":"WARN"`, `"level":"ERROR"`, `"level":"FATAL"`, `"level":"PANIC"`, `"level":"ERROR+12"`} {
		assert.Contains(t, out, want, "output missing level substring: %s", want)
	}
}

// User ReplaceAttr still runs after the level-name replacer (drop attr + trace
// level preserved).
func TestMergeHandlerOptions_chainsUserReplaceAttr(t *testing.T) {
	var buf bytes.Buffer
	opts := &slog.HandlerOptions{
		Level: rlog.LevelTrace,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == "drop" {
				return slog.Attr{}
			}
			return a
		},
	}
	logger := newTestLogger(t, &buf, opts)
	logger.Log(context.Background(), rlog.LevelTrace, "x", "drop", true, "keep", "y")

	out := buf.String()
	assert.NotContains(t, out, "drop", "user ReplaceAttr should drop key")
	assert.Contains(t, out, `"level":"TRACE"`)
	assert.Contains(t, out, "keep")
}

// Min level Error: Trace is below threshold, Panic is not.
func TestEnabled_ordering(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(t, &buf, &slog.HandlerOptions{Level: slog.LevelError})
	assert.False(t, logger.Enabled(context.Background(), rlog.LevelTrace), "Trace should be disabled when min is Error")
	assert.True(t, logger.Enabled(context.Background(), rlog.LevelPanic), "Panic should be enabled when min is Error")
}

func TestDefaultAttrs(t *testing.T) {
	t.Run("DebugAttrs", func(t *testing.T) {
		var buf bytes.Buffer
		logger := newTestLogger(t, &buf, &slog.HandlerOptions{Level: slog.LevelDebug})
		logger.DebugAttrs(context.Background(), "hello", slog.String("k", "v"))
		out := buf.String()
		assert.Contains(t, out, `"level":"DEBUG"`)
		assert.Contains(t, out, `"msg":"hello"`)
		assert.Contains(t, out, `"k":"v"`)
	})

	t.Run("InfoAttrs", func(t *testing.T) {
		var buf bytes.Buffer
		logger := newTestLogger(t, &buf, &slog.HandlerOptions{Level: slog.LevelInfo})
		logger.InfoAttrs(context.Background(), "hello", slog.String("k", "v"))
		out := buf.String()
		assert.Contains(t, out, `"level":"INFO"`)
		assert.Contains(t, out, `"msg":"hello"`)
		assert.Contains(t, out, `"k":"v"`)
	})

	t.Run("WarnAttrs", func(t *testing.T) {
		var buf bytes.Buffer
		logger := newTestLogger(t, &buf, &slog.HandlerOptions{Level: slog.LevelWarn})
		logger.WarnAttrs(context.Background(), "hello", slog.String("k", "v"))
		out := buf.String()
		assert.Contains(t, out, `"level":"WARN"`)
		assert.Contains(t, out, `"msg":"hello"`)
		assert.Contains(t, out, `"k":"v"`)
	})

	t.Run("ErrorAttrs", func(t *testing.T) {
		var buf bytes.Buffer
		logger := newTestLogger(t, &buf, &slog.HandlerOptions{Level: slog.LevelError})
		logger.ErrorAttrs(context.Background(), "hello", slog.String("k", "v"))
		out := buf.String()
		assert.Contains(t, out, `"level":"ERROR"`)
		assert.Contains(t, out, `"msg":"hello"`)
		assert.Contains(t, out, `"k":"v"`)
	})
}

func TestTrace(t *testing.T) {
	t.Run("Trace", func(t *testing.T) {
		var buf bytes.Buffer
		logger := newTestLogger(t, &buf, &slog.HandlerOptions{Level: rlog.LevelTrace})
		logger.Trace("hello", "k", "v")
		out := buf.String()
		assert.Contains(t, out, `"level":"TRACE"`)
		assert.Contains(t, out, `"msg":"hello"`)
		assert.Contains(t, out, `"k":"v"`)
	})

	t.Run("TraceContext", func(t *testing.T) {
		var buf bytes.Buffer
		logger := newTestLogger(t, &buf, &slog.HandlerOptions{Level: rlog.LevelTrace})
		logger.TraceContext(context.Background(), "hello", "k", "v")
		out := buf.String()
		assert.Contains(t, out, `"level":"TRACE"`)
		assert.Contains(t, out, `"msg":"hello"`)
		assert.Contains(t, out, `"k":"v"`)
	})

	t.Run("TraceAttrs", func(t *testing.T) {
		var buf bytes.Buffer
		logger := newTestLogger(t, &buf, &slog.HandlerOptions{Level: rlog.LevelTrace})
		logger.TraceAttrs(context.Background(), "hello", slog.String("k", "v"))
		out := buf.String()
		assert.Contains(t, out, `"level":"TRACE"`)
		assert.Contains(t, out, `"msg":"hello"`)
		assert.Contains(t, out, `"k":"v"`)
	})
}

func TestFatal(t *testing.T) {
	t.Run("Fatal", func(t *testing.T) {
		var buf bytes.Buffer
		var exited bool
		logger := newTestLogger(t, &buf, &slog.HandlerOptions{Level: rlog.LevelFatal})
		logger.SetExitFunc(func() { exited = true })
		defer logger.SetExitFunc(func() { os.Exit(1) })
		logger.Fatal("bye")
		assert.True(t, exited)
		out := buf.String()
		assert.Contains(t, out, `"level":"FATAL"`)
		assert.Contains(t, out, `"msg":"bye"`)
	})

	t.Run("FatalContext", func(t *testing.T) {
		var buf bytes.Buffer
		var exited bool
		logger := newTestLogger(t, &buf, &slog.HandlerOptions{Level: rlog.LevelFatal})
		logger.SetExitFunc(func() { exited = true })
		defer logger.SetExitFunc(func() { os.Exit(1) })
		logger.FatalContext(context.Background(), "bye")
		assert.True(t, exited)
		out := buf.String()
		assert.Contains(t, out, `"level":"FATAL"`)
		assert.Contains(t, out, `"msg":"bye"`)
	})

	t.Run("FatalAttrs", func(t *testing.T) {
		var buf bytes.Buffer
		var exited bool
		logger := newTestLogger(t, &buf, &slog.HandlerOptions{Level: rlog.LevelFatal})
		logger.SetExitFunc(func() { exited = true })
		defer logger.SetExitFunc(func() { os.Exit(1) })
		logger.FatalAttrs(context.Background(), "bye", slog.String("k", "v"))
		assert.True(t, exited)
		out := buf.String()
		assert.Contains(t, out, `"level":"FATAL"`)
		assert.Contains(t, out, `"msg":"bye"`)
		assert.Contains(t, out, `"k":"v"`)
	})
}

func TestPanic(t *testing.T) {
	t.Run("Panic", func(t *testing.T) {
		var buf bytes.Buffer
		logger := newTestLogger(t, &buf, &slog.HandlerOptions{Level: rlog.LevelPanic})
		assert.PanicsWithValue(t, "boom", func() {
			logger.Panic("boom")
		})
		out := buf.String()
		assert.Contains(t, out, `"level":"PANIC"`)
		assert.Contains(t, out, `"msg":"boom"`)
	})

	t.Run("PanicContext", func(t *testing.T) {
		var buf bytes.Buffer
		logger := newTestLogger(t, &buf, &slog.HandlerOptions{Level: rlog.LevelPanic})
		assert.PanicsWithValue(t, "ctx", func() {
			logger.PanicContext(context.Background(), "ctx")
		})
		out := buf.String()
		assert.Contains(t, out, `"level":"PANIC"`)
		assert.Contains(t, out, `"msg":"ctx"`)
	})

	t.Run("PanicAttrs", func(t *testing.T) {
		var buf bytes.Buffer
		logger := newTestLogger(t, &buf, &slog.HandlerOptions{Level: rlog.LevelPanic})
		assert.PanicsWithValue(t, "attrs", func() {
			logger.PanicAttrs(context.Background(), "attrs", slog.String("k", "v"))
		})
		out := buf.String()
		assert.Contains(t, out, `"level":"PANIC"`)
		assert.Contains(t, out, `"msg":"attrs"`)
		assert.Contains(t, out, `"k":"v"`)
	})
}

// Decoded JSON has string level "TRACE" (not a numeric slog offset).
func TestJSON_shape(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(t, &buf, &slog.HandlerOptions{Level: rlog.LevelTrace})
	logger.Log(context.Background(), rlog.LevelTrace, "x")

	var m map[string]any
	assert.Ok(t, json.Unmarshal(buf.Bytes(), &m))
	assert.Equal(t, "TRACE", m["level"])
}

// Default and SetDefault must remain safe under heavy concurrent use (run with -race).
// The race detector flags unsafe access to the package-global logger; atomics verify
// that every goroutine ran its full loop and that logging paths executed as expected.
func TestDefaultSetDefault_concurrent(t *testing.T) {
	// Enough goroutines and iterations to interleave real workloads; poolSize > 1 so
	// SetDefault swaps among different logger instances rather than one repeated value.
	const (
		goroutines = 128
		iterations = 400
		poolSize   = 16
	)

	prev := rlog.Default()
	t.Cleanup(func() { rlog.SetDefault(prev) })

	pool := make([]*rlog.Logger, poolSize)
	for i := range pool {
		h := slog.NewJSONHandler(io.Discard, rlog.MergeWithCustomLevels(&slog.HandlerOptions{Level: rlog.LevelTrace}))
		pool[i] = rlog.New(slog.New(h))
	}

	// Each inner-loop iteration runs exactly one switch arm; (id+n)%4 cycles all four
	// equally over iterations, so per goroutine each arm runs iterations/4 times.
	perG := iterations / 4
	wantIterations := int64(goroutines * iterations)
	wantLog := int64(goroutines * perG * 3) // only arms 1–3 log; each logs once per hit

	var wg sync.WaitGroup
	var iterationsDone, logOps, nilishLoggers atomic.Int64
	wg.Add(goroutines)
	for g := range goroutines {
		go func(id int) {
			defer wg.Done()
			for n := range iterations {
				// Four interleavings: writer-only, read+log on *Logger, package-level
				// Info (which calls Default again internally), and set-then-read-then-log.
				switch (id + n) % 4 {
				case 0:
					rlog.SetDefault(pool[(id+n)%poolSize])
				case 1:
					l := rlog.Default()
					noteNilishLogger(l, &nilishLoggers)
					if l != nil && l.Logger != nil {
						l.Info("concurrent")
						logOps.Add(1)
					}
				case 2:
					l := rlog.Default()
					noteNilishLogger(l, &nilishLoggers)
					rlog.Info("concurrent-global")
					logOps.Add(1)
				case 3:
					rlog.SetDefault(pool[(id*3+n)%poolSize])
					l := rlog.Default()
					noteNilishLogger(l, &nilishLoggers)
					if l != nil && l.Logger != nil {
						l.Info("after-set")
						logOps.Add(1)
					}
				}
				iterationsDone.Add(1)
			}
		}(g)
	}
	wg.Wait()

	// nilishLoggers: regression sentinel for a broken or torn default logger.
	// iterationsDone: every inner-loop body completed (no panic, no stuck goroutine).
	// logOps: arms that should emit a log did so the expected number of times.
	assert.Equal(t, int64(0), nilishLoggers.Load(), "nil *Logger or nil embedded slog.Logger under concurrency")
	assert.Equal(t, wantIterations, iterationsDone.Load())
	assert.Equal(t, wantLog, logOps.Load())

	// Smoke check: global default is still usable after concurrent churn.
	rlog.Default().Info("post-stress")
}

// Global package functions must log through [rlog.SetDefault], not the library default.
func TestGlobalFunctionsUseSetDefaultLogger(t *testing.T) {
	var buf bytes.Buffer
	opts := &slog.HandlerOptions{Level: rlog.LevelTrace}
	inner := slog.New(slog.NewTextHandler(&buf, rlog.MergeWithCustomLevels(opts)))
	lg := rlog.New(inner)
	var exited bool
	lg.SetExitFunc(func() { exited = true })

	prev := rlog.Default()
	t.Cleanup(func() { rlog.SetDefault(prev) })
	rlog.SetDefault(lg)

	ctx := context.Background()

	type gCase struct {
		name  string
		run   func()
		want  []string
		fatal bool
		panic bool
	}

	cases := []gCase{
		{"Log", func() { rlog.Log(ctx, slog.LevelInfo, "log-msg") }, []string{"INFO", "log-msg"}, false, false},
		{"LogAttrs", func() {
			rlog.LogAttrs(ctx, slog.LevelWarn, "la-msg", slog.String("k", "v"))
		}, []string{"WARN", "la-msg", "k=v"}, false, false},

		{"Trace", func() { rlog.Trace("trace-msg") }, []string{"TRACE", "trace-msg"}, false, false},
		{"Debug", func() { rlog.Debug("debug-msg") }, []string{"DEBUG", "debug-msg"}, false, false},
		{"Info", func() { rlog.Info("info-msg") }, []string{"INFO", "info-msg"}, false, false},
		{"Warn", func() { rlog.Warn("warn-msg") }, []string{"WARN", "warn-msg"}, false, false},
		{"Error", func() { rlog.Error("error-msg") }, []string{"ERROR", "error-msg"}, false, false},
		{"Fatal", func() { rlog.Fatal("fatal-msg") }, []string{"FATAL", "fatal-msg"}, true, false},
		{"Panic", func() { rlog.Panic("panic-msg") }, []string{"PANIC", "panic-msg"}, false, true},

		{"TraceContext", func() { rlog.TraceContext(ctx, "tc") }, []string{"TRACE", "tc"}, false, false},
		{"DebugContext", func() { rlog.DebugContext(ctx, "dc") }, []string{"DEBUG", "dc"}, false, false},
		{"InfoContext", func() { rlog.InfoContext(ctx, "ic") }, []string{"INFO", "ic"}, false, false},
		{"WarnContext", func() { rlog.WarnContext(ctx, "wc") }, []string{"WARN", "wc"}, false, false},
		{"ErrorContext", func() { rlog.ErrorContext(ctx, "ec") }, []string{"ERROR", "ec"}, false, false},
		{"FatalContext", func() { rlog.FatalContext(ctx, "fc") }, []string{"FATAL", "fc"}, true, false},
		{"PanicContext", func() { rlog.PanicContext(ctx, "pc") }, []string{"PANIC", "pc"}, false, true},

		{"TraceAttrs", func() { rlog.TraceAttrs(ctx, "ta", slog.String("a", "1")) }, []string{"TRACE", "ta", "a=1"}, false, false},
		{"DebugAttrs", func() { rlog.DebugAttrs(ctx, "da", slog.String("a", "1")) }, []string{"DEBUG", "da", "a=1"}, false, false},
		{"InfoAttrs", func() { rlog.InfoAttrs(ctx, "ia", slog.String("a", "1")) }, []string{"INFO", "ia", "a=1"}, false, false},
		{"WarnAttrs", func() { rlog.WarnAttrs(ctx, "wa", slog.String("a", "1")) }, []string{"WARN", "wa", "a=1"}, false, false},
		{"ErrorAttrs", func() { rlog.ErrorAttrs(ctx, "ea", slog.String("a", "1")) }, []string{"ERROR", "ea", "a=1"}, false, false},
		{"FatalAttrs", func() { rlog.FatalAttrs(ctx, "fa", slog.String("a", "1")) }, []string{"FATAL", "fa", "a=1"}, true, false},
		{"PanicAttrs", func() { rlog.PanicAttrs(ctx, "pa", slog.String("a", "1")) }, []string{"PANIC", "pa", "a=1"}, false, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf.Reset()
			exited = false

			switch {
			case tc.panic:
				var sawPanic bool
				func() {
					defer func() {
						if recover() != nil {
							sawPanic = true
						}
					}()
					tc.run()
				}()
				assert.True(t, sawPanic, "expected panic from %s", tc.name)
			case tc.fatal:
				tc.run()
				assert.True(t, exited, "expected exit hook from %s", tc.name)
			default:
				tc.run()
			}

			out := buf.String()
			for _, w := range tc.want {
				assert.Contains(t, out, w)
			}
		})
	}
}

//=============================================================================
// Helpers
//=============================================================================

func newTestLogger(t *testing.T, buf *bytes.Buffer, opts *slog.HandlerOptions) *rlog.Logger {
	t.Helper()
	inner := slog.New(slog.NewJSONHandler(buf, rlog.MergeWithCustomLevels(opts)))
	return rlog.New(inner)
}

// noteNilishLogger increments cnt if l or its embedded [*slog.Logger] is nil.
func noteNilishLogger(l *rlog.Logger, cnt *atomic.Int64) {
	if l == nil {
		cnt.Add(1)
		return
	}
	if l.Logger == nil {
		cnt.Add(1)
	}
}

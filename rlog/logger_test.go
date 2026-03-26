package rlog_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/rlog"
)

// Logger *Attrs helpers emit JSON with expected level and structured fields.
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

// Trace, TraceContext, and TraceAttrs write at LevelTrace when enabled.
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

// Fatal, FatalContext, and FatalAttrs write at LevelFatal when enabled and run the fatal hook.
func TestFatal(t *testing.T) {
	t.Run("Fatal", func(t *testing.T) {
		var buf bytes.Buffer
		var exited bool
		logger := newTestLogger(t, &buf, &slog.HandlerOptions{Level: rlog.LevelFatal})

		rlog.SetFatalHook(func() { exited = true })
		t.Cleanup(func() { rlog.SetFatalHook(nil) })

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

		rlog.SetFatalHook(func() { exited = true })
		t.Cleanup(func() { rlog.SetFatalHook(nil) })

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

		rlog.SetFatalHook(func() { exited = true })
		t.Cleanup(func() { rlog.SetFatalHook(nil) })

		logger.FatalAttrs(context.Background(), "bye", slog.String("k", "v"))
		assert.True(t, exited)
		out := buf.String()
		assert.Contains(t, out, `"level":"FATAL"`)
		assert.Contains(t, out, `"msg":"bye"`)
		assert.Contains(t, out, `"k":"v"`)
	})
}

// Panic, PanicContext, and PanicAttrs write at LevelPanic when enabled and panic with the message.
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

// Concurrent SetDefault, Default, and package-level logging (run with -race).
func TestDefaultSetDefault_concurrent(t *testing.T) {
	// Pool of loggers to rotate under SetDefault.
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

	// Expected counts: each goroutine hits each switch arm equally; only three arms log.
	perG := iterations / 4
	wantIterations := int64(goroutines * iterations)
	wantLog := int64(goroutines * perG * 3)

	var wg sync.WaitGroup
	var iterationsDone, logOps atomic.Int64
	wg.Add(goroutines)
	for g := range goroutines {
		go func(id int) {
			defer wg.Done()
			for n := range iterations {
				switch (id + n) % 4 {
				case 0: // swap global only
					rlog.SetDefault(pool[(id+n)%poolSize])
				case 1: // read default, method log
					rlog.Default().Info("concurrent")
					logOps.Add(1)
				case 2: // read default, package-level log
					_ = rlog.Default()
					rlog.Info("concurrent-global")
					logOps.Add(1)
				case 3: // swap then read and log
					rlog.SetDefault(pool[(id*3+n)%poolSize])
					rlog.Default().Info("after-set")
					logOps.Add(1)
				}
				iterationsDone.Add(1)
			}
		}(g)
	}
	wg.Wait()

	assert.Equal(t, wantIterations, iterationsDone.Load())
	assert.Equal(t, wantLog, logOps.Load())

	rlog.Default().Info("post-stress")
}

// Package-level rlog.* functions delegate to SetDefault's logger.
func TestGlobalFunctionsUseSetDefaultLogger(t *testing.T) {
	var buf bytes.Buffer
	opts := &slog.HandlerOptions{Level: rlog.LevelTrace}
	inner := slog.New(slog.NewTextHandler(&buf, rlog.MergeWithCustomLevels(opts)))
	lg := rlog.New(inner)

	rlog.SetFatalHook(func() {})
	t.Cleanup(func() { rlog.SetFatalHook(nil) })

	prev := rlog.Default()
	t.Cleanup(func() { rlog.SetDefault(prev) })

	// Set the default logger to the one we created and verify that it matches [slog.Default].
	rlog.SetDefault(lg)
	assert.Equal(t, rlog.Default().Logger, slog.Default(), "slog.Default must match rlog's embedded logger after SetDefault")

	ctx := context.Background()

	type gCase struct {
		name  string
		run   func()
		want  []string
		panic bool
	}

	cases := []gCase{
		{"Log", func() { rlog.Log(ctx, slog.LevelInfo, "log-msg") }, []string{"INFO", "log-msg"}, false},
		{"LogAttrs", func() {
			rlog.LogAttrs(ctx, slog.LevelWarn, "la-msg", slog.String("k", "v"))
		}, []string{"WARN", "la-msg", "k=v"}, false},

		{"Trace", func() { rlog.Trace("trace-msg") }, []string{"TRACE", "trace-msg"}, false},
		{"Debug", func() { rlog.Debug("debug-msg") }, []string{"DEBUG", "debug-msg"}, false},
		{"Info", func() { rlog.Info("info-msg") }, []string{"INFO", "info-msg"}, false},
		{"Warn", func() { rlog.Warn("warn-msg") }, []string{"WARN", "warn-msg"}, false},
		{"Error", func() { rlog.Error("error-msg") }, []string{"ERROR", "error-msg"}, false},
		{"Fatal", func() { rlog.Fatal("fatal-msg") }, []string{"FATAL", "fatal-msg"}, false},
		{"Panic", func() { rlog.Panic("panic-msg") }, []string{"PANIC", "panic-msg"}, true},

		{"TraceContext", func() { rlog.TraceContext(ctx, "tc") }, []string{"TRACE", "tc"}, false},
		{"DebugContext", func() { rlog.DebugContext(ctx, "dc") }, []string{"DEBUG", "dc"}, false},
		{"InfoContext", func() { rlog.InfoContext(ctx, "ic") }, []string{"INFO", "ic"}, false},
		{"WarnContext", func() { rlog.WarnContext(ctx, "wc") }, []string{"WARN", "wc"}, false},
		{"ErrorContext", func() { rlog.ErrorContext(ctx, "ec") }, []string{"ERROR", "ec"}, false},
		{"FatalContext", func() { rlog.FatalContext(ctx, "fc") }, []string{"FATAL", "fc"}, false},
		{"PanicContext", func() { rlog.PanicContext(ctx, "pc") }, []string{"PANIC", "pc"}, true},

		{"TraceAttrs", func() { rlog.TraceAttrs(ctx, "ta", slog.String("a", "1")) }, []string{"TRACE", "ta", "a=1"}, false},
		{"DebugAttrs", func() { rlog.DebugAttrs(ctx, "da", slog.String("a", "1")) }, []string{"DEBUG", "da", "a=1"}, false},
		{"InfoAttrs", func() { rlog.InfoAttrs(ctx, "ia", slog.String("a", "1")) }, []string{"INFO", "ia", "a=1"}, false},
		{"WarnAttrs", func() { rlog.WarnAttrs(ctx, "wa", slog.String("a", "1")) }, []string{"WARN", "wa", "a=1"}, false},
		{"ErrorAttrs", func() { rlog.ErrorAttrs(ctx, "ea", slog.String("a", "1")) }, []string{"ERROR", "ea", "a=1"}, false},
		{"FatalAttrs", func() { rlog.FatalAttrs(ctx, "fa", slog.String("a", "1")) }, []string{"FATAL", "fa", "a=1"}, false},
		{"PanicAttrs", func() { rlog.PanicAttrs(ctx, "pa", slog.String("a", "1")) }, []string{"PANIC", "pa", "a=1"}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf.Reset()

			// If the test case expects a panic, verify that a panic was actually raised.
			if tc.panic {
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
			} else {
				tc.run()
			}

			// Verify the output contains the expected substrings.
			out := buf.String()
			for _, w := range tc.want {
				assert.Contains(t, out, w)
			}
		})
	}
}

// rlog.SetDefault updates slog.Default to the same underlying logger.
func TestSlogSetDefault_alignedWithRlog(t *testing.T) {
	var buf bytes.Buffer
	opts := rlog.MergeWithCustomLevels(&slog.HandlerOptions{Level: slog.LevelInfo})
	lg := rlog.New(slog.New(slog.NewJSONHandler(&buf, opts)))
	prev := rlog.Default()
	t.Cleanup(func() { rlog.SetDefault(prev) })

	rlog.SetDefault(lg)
	assert.Equal(t, rlog.Default().Logger, slog.Default(), "slog.Default must match rlog's embedded logger after SetDefault")

	slog.Info("via-slog")
	assert.Contains(t, buf.String(), "via-slog")
}

//=============================================================================
// Helper Functions
//=============================================================================

// newTestLogger wraps JSONHandler with MergeWithCustomLevels so TRACE/FATAL/PANIC levels work.
func newTestLogger(t *testing.T, buf *bytes.Buffer, opts *slog.HandlerOptions) *rlog.Logger {
	t.Helper()
	inner := slog.New(slog.NewJSONHandler(buf, rlog.MergeWithCustomLevels(opts)))
	return rlog.New(inner)
}

package rlog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"testing"

	"go.rtnl.ai/endeavor/pkg/rlog"
	"go.rtnl.ai/x/assert"
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
		assert.True(t, strings.Contains(out, want), "output missing level substring")
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
	assert.False(t, strings.Contains(out, "drop"), "user ReplaceAttr should drop key")
	assert.True(t, strings.Contains(out, `"level":"TRACE"`))
	assert.True(t, strings.Contains(out, "keep"))
}

// Min level Error: Trace is below threshold, Panic is not.
func TestEnabled_ordering(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(t, &buf, &slog.HandlerOptions{Level: slog.LevelError})
	assert.False(t, logger.Enabled(context.Background(), rlog.LevelTrace), "Trace should be disabled when min is Error")
	assert.True(t, logger.Enabled(context.Background(), rlog.LevelPanic), "Panic should be enabled when min is Error")
}

func TestTrace(t *testing.T) {
	t.Run("Trace", func(t *testing.T) {
		var buf bytes.Buffer
		logger := newTestLogger(t, &buf, &slog.HandlerOptions{Level: rlog.LevelTrace})
		logger.Trace("hello", "k", "v")
		out := buf.String()
		assert.True(t, strings.Contains(out, `"msg":"hello"`))
		assert.True(t, strings.Contains(out, `"k":"v"`))
	})

	t.Run("TraceContext", func(t *testing.T) {
		var buf bytes.Buffer
		logger := newTestLogger(t, &buf, &slog.HandlerOptions{Level: rlog.LevelTrace})
		logger.TraceContext(context.Background(), "hello", "k", "v")
		out := buf.String()
		assert.True(t, strings.Contains(out, `"msg":"hello"`))
		assert.True(t, strings.Contains(out, `"k":"v"`))
	})

	t.Run("TraceAttrs", func(t *testing.T) {
		var buf bytes.Buffer
		logger := newTestLogger(t, &buf, &slog.HandlerOptions{Level: rlog.LevelTrace})
		logger.TraceAttrs(context.Background(), "hello", slog.String("k", "v"))
		out := buf.String()
		assert.True(t, strings.Contains(out, `"msg":"hello"`))
		assert.True(t, strings.Contains(out, `"k":"v"`))
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
		assert.True(t, strings.Contains(buf.String(), `"msg":"bye"`))
	})

	t.Run("FatalContext", func(t *testing.T) {
		var buf bytes.Buffer
		var exited bool
		logger := newTestLogger(t, &buf, &slog.HandlerOptions{Level: rlog.LevelFatal})
		logger.SetExitFunc(func() { exited = true })
		defer logger.SetExitFunc(func() { os.Exit(1) })
		logger.FatalContext(context.Background(), "bye")
		assert.True(t, exited)
		assert.True(t, strings.Contains(buf.String(), `"msg":"bye"`))
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
		assert.True(t, strings.Contains(out, `"msg":"bye"`))
		assert.True(t, strings.Contains(out, `"k":"v"`))
	})
}

func TestPanic(t *testing.T) {
	t.Run("Panic", func(t *testing.T) {
		var buf bytes.Buffer
		logger := newTestLogger(t, &buf, &slog.HandlerOptions{Level: rlog.LevelPanic})
		assertPanicsWithValue(t, "boom", func() {
			logger.Panic("boom")
		})
		assert.True(t, strings.Contains(buf.String(), `"msg":"boom"`))
	})

	t.Run("PanicContext", func(t *testing.T) {
		var buf bytes.Buffer
		logger := newTestLogger(t, &buf, &slog.HandlerOptions{Level: rlog.LevelPanic})
		assertPanicsWithValue(t, "ctx", func() {
			logger.PanicContext(context.Background(), "ctx")
		})
		assert.True(t, strings.Contains(buf.String(), `"msg":"ctx"`))
	})

	t.Run("PanicAttrs", func(t *testing.T) {
		var buf bytes.Buffer
		logger := newTestLogger(t, &buf, &slog.HandlerOptions{Level: rlog.LevelPanic})
		assertPanicsWithValue(t, "attrs", func() {
			logger.PanicAttrs(context.Background(), "attrs", slog.String("k", "v"))
		})
		out := buf.String()
		assert.True(t, strings.Contains(out, `"msg":"attrs"`))
		assert.True(t, strings.Contains(out, `"k":"v"`))
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

func newTestLogger(t *testing.T, buf *bytes.Buffer, opts *slog.HandlerOptions) *rlog.Logger {
	t.Helper()
	inner := slog.New(slog.NewJSONHandler(buf, rlog.MergeWithCustomLevels(opts)))
	return rlog.New(inner)
}

func assertPanicsWithValue(t *testing.T, expected any, fn func()) {
	t.Helper()
	defer func() {
		recovered := recover()
		assert.NotNil(t, recovered, "expected panic")
		assert.Equal(t, expected, recovered)
	}()
	fn()
}

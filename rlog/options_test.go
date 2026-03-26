package rlog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/rlog"
)

// Verifies MergeWithCustomLevels maps custom levels to TRACE/FATAL/PANIC strings and leaves other levels as slog defaults.
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

// User ReplaceAttr still runs after the level-name replacer (drop attr + trace level preserved).
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

// Decoded JSON has string level "TRACE" (not a numeric slog offset).
func TestJSON_shape(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(t, &buf, &slog.HandlerOptions{Level: rlog.LevelTrace})
	logger.Log(context.Background(), rlog.LevelTrace, "x")

	var m map[string]any
	assert.Ok(t, json.Unmarshal(buf.Bytes(), &m))
	assert.Equal(t, "TRACE", m["level"])
}

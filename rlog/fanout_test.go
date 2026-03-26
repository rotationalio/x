package rlog_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"testing/slogtest"
	"time"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/rlog"
)

// Each sink receives the same logical record when levels allow.
func TestFanOut_Handle_clonesToEachSink(t *testing.T) {
	var a, b bytes.Buffer
	ha := slog.NewJSONHandler(&a, rlog.MergeWithCustomLevels(&slog.HandlerOptions{Level: slog.LevelInfo}))
	hb := slog.NewJSONHandler(&b, rlog.MergeWithCustomLevels(&slog.HandlerOptions{Level: slog.LevelInfo}))
	f := rlog.NewFanOut(ha, hb)

	ctx := context.Background()
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "hello", 0)
	assert.Ok(t, f.Handle(ctx, r))

	assert.Contains(t, a.String(), "hello")
	assert.Contains(t, b.String(), "hello")
}

// Enabled is true if any child is enabled for the level.
func TestFanOut_Enabled_OR(t *testing.T) {
	var quiet, loud bytes.Buffer
	hQuiet := slog.NewJSONHandler(&quiet, &slog.HandlerOptions{Level: slog.LevelError})
	hLoud := slog.NewJSONHandler(&loud, &slog.HandlerOptions{Level: slog.LevelDebug})
	f := rlog.NewFanOut(hQuiet, hLoud)
	ctx := context.Background()

	assert.True(t, f.Enabled(ctx, slog.LevelInfo), "loud child accepts Info")
	assert.False(t, f.Enabled(ctx, rlog.LevelTrace), "neither accepts Trace by default opts")
}

// Handle skips children whose minimum level is above the record (like slog.MultiHandler).
func TestFanOut_Handle_skipsDisabledChildren(t *testing.T) {
	var quiet, loud bytes.Buffer
	hQuiet := slog.NewJSONHandler(&quiet, &slog.HandlerOptions{Level: slog.LevelError})
	hLoud := slog.NewJSONHandler(&loud, &slog.HandlerOptions{Level: slog.LevelDebug})
	f := rlog.NewFanOut(hQuiet, hLoud)

	ctx := context.Background()
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "hello", 0)
	assert.Ok(t, f.Handle(ctx, r))

	assert.Equal(t, "", strings.TrimSpace(quiet.String()), "Error-only sink must not receive Info")
	assert.Contains(t, loud.String(), "hello")
}

// WithGroup on the fan-out applies to every child.
func TestFanOut_WithGroup_propagates(t *testing.T) {
	var a, b bytes.Buffer
	ha := slog.NewJSONHandler(&a, rlog.MergeWithCustomLevels(&slog.HandlerOptions{Level: slog.LevelInfo}))
	hb := slog.NewJSONHandler(&b, rlog.MergeWithCustomLevels(&slog.HandlerOptions{Level: slog.LevelInfo}))
	f := rlog.NewFanOut(ha, hb).WithGroup("outer").(*rlog.FanOut)

	ctx := context.Background()
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.String("k", "v"))
	assert.Ok(t, f.Handle(ctx, r))

	for _, out := range []string{a.String(), b.String()} {
		assert.Contains(t, out, `"outer":`)
		assert.Contains(t, out, `"k":"v"`)
	}
}

// Single-child fan-out should satisfy slog's handler test suite.
func TestFanOut_slogtest_singleChild(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	fan := rlog.NewFanOut(h)

	err := slogtest.TestHandler(fan, func() []map[string]any {
		var maps []map[string]any
		for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
			if line == "" {
				continue
			}
			m, e := rlog.ParseJSONLine(line)
			assert.Ok(t, e)
			maps = append(maps, m)
		}
		return maps
	})
	assert.Ok(t, err)
}

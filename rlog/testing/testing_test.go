package testing_test

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"testing/slogtest"
	"time"

	"go.rtnl.ai/x/assert"
	rlogtesting "go.rtnl.ai/x/rlog/testing"
)

// CapturingTestHandler should pass the standard slog handler tests.
func TestCapturingTestHandler_slogtest(t *testing.T) {
	cap := rlogtesting.NewCapturingTestHandler(nil)
	err := slogtest.TestHandler(cap, func() []map[string]any {
		maps, err := cap.ResultMaps()
		assert.Ok(t, err)
		return maps
	})
	assert.Ok(t, err)
}

// Groups nest attrs in JSON output and ResultMaps sees nested maps.
func TestCapturingTestHandler_groupNesting(t *testing.T) {
	h := rlogtesting.NewCapturingTestHandler(nil).WithGroup("g").(*rlogtesting.CapturingTestHandler)
	h = h.WithAttrs([]slog.Attr{slog.String("inside", "yes")}).(*rlogtesting.CapturingTestHandler)

	ctx := context.Background()
	assert.Ok(t, h.Handle(ctx, slog.NewRecord(time.Now(), slog.LevelInfo, "hi", 0)))

	lines := h.Lines()
	assert.Equal(t, 1, len(lines))
	m := rlogtesting.MustParseJSONLine(lines[0])
	g, ok := m["g"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "yes", g["inside"])
}

// ParseJSONLine and MustParseJSONLine round-trip a simple object.
func TestParseJSONLine_helpers(t *testing.T) {
	const line = `{"level":"INFO","msg":"x","k":1}`
	m, err := rlogtesting.ParseJSONLine(line)
	assert.Ok(t, err)
	assert.Equal(t, "INFO", m["level"])
	assert.Equal(t, float64(1), m["k"])
	assert.Equal(t, m, rlogtesting.MustParseJSONLine(line))
}

// NewCapturingTestHandler links to [testing.TB] (only *testing.T / B / F satisfy TB outside stdlib).
// Derived handlers still share capture state; slogtest exercises WithAttrs/WithGroup with tb set.
func TestCapturingTestHandler_slogtest_withTB(t *testing.T) {
	cap := rlogtesting.NewCapturingTestHandler(t)
	err := slogtest.TestHandler(cap, func() []map[string]any {
		maps, err := cap.ResultMaps()
		assert.Ok(t, err)
		return maps
	})
	assert.Ok(t, err)
}

// Concurrent Handle calls must keep the same index aligned in records and lines.
func TestCapturingTestHandler_concurrentHandle_recordsMatchLines(t *testing.T) {
	h := rlogtesting.NewCapturingTestHandler(nil)
	ctx := context.Background()
	const n = 64
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			msg := fmt.Sprintf("msg-%d", i)
			assert.Ok(t, h.Handle(ctx, slog.NewRecord(time.Now(), slog.LevelInfo, msg, 0)))
		}(i)
	}
	wg.Wait()

	recs, lines := h.RecordsAndLines()
	assert.Equal(t, len(recs), len(lines))
	assert.Equal(t, n, len(recs))
	for i := range recs {
		assert.Contains(t, lines[i], recs[i].Message)
	}
}

// Derived handlers share the same capture buffers.
func TestCapturingTestHandler_derivedSharesLines(t *testing.T) {
	h := rlogtesting.NewCapturingTestHandler(nil)
	ctx := context.Background()
	assert.Ok(t, h.Handle(ctx, slog.NewRecord(time.Now(), slog.LevelInfo, "one", 0)))

	h2 := h.WithAttrs([]slog.Attr{slog.String("k", "v")}).(*rlogtesting.CapturingTestHandler)
	assert.Ok(t, h2.Handle(ctx, slog.NewRecord(time.Now(), slog.LevelWarn, "two", 0)))

	lines := h.Lines()
	assert.Equal(t, 2, len(lines))
	assert.True(t, strings.Contains(lines[0], `"msg":"one"`))
	assert.True(t, strings.Contains(lines[1], `"k":"v"`))
	assert.True(t, strings.Contains(lines[1], `"msg":"two"`))
}

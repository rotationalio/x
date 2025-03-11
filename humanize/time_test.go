package humanize_test

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/humanize"
)

func TestTime(t *testing.T) {
	tests := []struct {
		time   time.Time
		suffix string
	}{
		{time.Now().Add(-265 * time.Second), "ago"},
		{time.Now().Add(265 * time.Second), "from now"},
	}

	for i, tc := range tests {
		dur := humanize.Time(tc.time)
		assert.True(t, strings.HasSuffix(dur, tc.suffix), "test case %d failed", i)
	}
}

func TestRelativeTime(t *testing.T) {
	t1 := time.Date(2021, 1, 21, 21, 21, 21, 0, time.UTC)
	durations := []struct {
		dur time.Duration
		fmt string
	}{
		{1500 * time.Millisecond, "1 second %s"},
		{31527 * time.Millisecond, "31 seconds %s"},
		{70 * time.Second, "1 minute %s"},
		{831 * time.Second, "13 minutes %s"},
		{98 * time.Minute, "1 hour %s"},
		{192 * time.Minute, "3 hours %s"},
		{32 * time.Hour, "1 day %s"},
		{98 * time.Hour, "4 days %s"},
		{9 * humanize.Day, "1 week %s"},
		{18 * humanize.Day, "2 weeks %s"},
		{6 * humanize.Week, "1 month %s"},
		{36 * humanize.Week, "8 months %s"},
		{14 * humanize.Month, "1 year %s"},
		{69 * humanize.Month, "5 years %s"},
		{180 * humanize.Year, "a long while %s"},
		{time.Duration(math.MaxInt64), "a long while %s"},
	}

	t.Run("Since", func(t *testing.T) {
		assert.Equal(t, "now", humanize.Since(t1, t1, "ago"), "now test 1 failed")
		assert.Equal(t, "now", humanize.Since(t1.Add(-500*time.Millisecond), t1, "ago"), "now test 2 failed")

		for i, tc := range durations {
			t0 := t1.Add(-1 * tc.dur)
			assert.Equal(t, fmt.Sprintf(tc.fmt, "ago"), humanize.Since(t0, t1, "ago"), "test case %d failed", i)
		}
	})

	t.Run("Until", func(t *testing.T) {
		assert.Equal(t, "now", humanize.Since(t1, t1, "from now"), "now test 1 failed")
		assert.Equal(t, "now", humanize.Since(t1.Add(500*time.Millisecond), t1, "from now"), "now test 2 failed")

		for i, tc := range durations {
			t0 := t1.Add(tc.dur)
			assert.Equal(t, fmt.Sprintf(tc.fmt, "from now"), humanize.Until(t0, t1, "from now"), "test case %d failed", i)
		}
	})
}

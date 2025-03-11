package humanize

import (
	"fmt"
	"math"
	"sort"
	"time"
)

const (
	Day      = 24 * time.Hour
	Week     = 7 * Day
	Month    = 30 * Day
	Year     = 12 * Month
	LongTime = 37 * Year
)

// Format the time into a relative string from now. E.g. "5 minutes ago" or "5 minutes from now".
// Inspiration from: https://github.com/dustin/go-humanize/blob/master/times.go
func Time(ts time.Time) string {
	return RelativeTime(ts, time.Now(), "ago", "from now", defaultMagnitudes)
}

// A TimePeriod struct contains a relative time point at which the relative duration
// format will switch to a new format string. E.g. if the duration (D) is 1 minute,
// then we would express a relative duration as "x seconds ago/from now" for any
// duration within the time period.
//
// The format string must contain a single %s verb, which will be replaced with the
// appropriate time direction (e.g. "ago" or "from now") and a %d which will be replaced
// with the quantity of time periods.
type TimePeriod struct {
	D      time.Duration // The length of the time period
	Format string        // The format string to use for the time period
	Den    time.Duration // The denominator to use to determine the number of periods
}

var defaultMagnitudes = []TimePeriod{
	{time.Second, "now", time.Second},
	{2 * time.Second, "1 second %s", 1},
	{time.Minute, "%d seconds %s", time.Second},
	{2 * time.Minute, "1 minute %s", 1},
	{time.Hour, "%d minutes %s", time.Minute},
	{2 * time.Hour, "1 hour %s", 1},
	{Day, "%d hours %s", time.Hour},
	{2 * Day, "1 day %s", 1},
	{Week, "%d days %s", Day},
	{2 * Week, "1 week %s", 1},
	{Month, "%d weeks %s", Week},
	{2 * Month, "1 month %s", 1},
	{Year, "%d months %s", Month},
	{18 * Month, "1 year %s", 1},
	{2 * Year, "2 years %s", 1},
	{LongTime, "%d years %s", Year},
	{math.MaxInt64, "a long while %s", 1},
}

// Since returns a string representing the time elapsed between t0 and t1. You must
// also pass in a label for the time period, e.g. "ago" or "earlier" to describe the
// amount of time that has passed. If t1 is before t0, you should use Until.
func Since(t0, t1 time.Time, label string) string {
	return RelativeTime(t0, t1, label, "", defaultMagnitudes)
}

// Until returns a string representing the time that will elapse between t0 and t1. You
// must also pass in a label for the time period, e.g. "from now" or "later" to describe
// the amount of time that will pass. If t1 is before t0, you should use Since.
func Until(t0, t1 time.Time, label string) string {
	return RelativeTime(t0, t1, "", label, defaultMagnitudes)
}

func RelativeTime(t0, t1 time.Time, earlier, later string, periods []TimePeriod) string {
	// Determine which the direction of time based on the order of t0 and t1
	var (
		diff  time.Duration
		label string
	)

	if t1.After(t0) {
		diff = t1.Sub(t0)
		label = earlier
	} else {
		diff = t0.Sub(t1)
		label = later
	}

	// Find the period that best describes the time difference
	n := sort.Search(len(periods), func(i int) bool {
		return periods[i].D > diff
	})

	// If the diff is greater than any of the periods, use the last period
	if n >= len(periods) {
		n = len(periods) - 1
	}

	// Prepare the format string
	mag := periods[n]
	args := []interface{}{}

	// Check the characters for each format string to order the arguments correctly
	escaped := false
	for _, char := range mag.Format {
		if char == '%' {
			escaped = true
			continue
		}

		if escaped {
			switch char {
			case 's':
				args = append(args, label)
			case 'd':
				args = append(args, diff/mag.Den)
			}
			escaped = false
		}
	}

	return fmt.Sprintf(mag.Format, args...)
}

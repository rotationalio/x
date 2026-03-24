package rlog

import (
	"log/slog"
)

// MergeWithCustomLevels returns a copy of opts (or a new [slog.HandlerOptions]
// if opts is nil) whose ReplaceAttr first maps [LevelTrace], [LevelFatal], and
// [LevelPanic] to the strings [LevelTraceKey], [LevelFatalKey], and [LevelPanicKey],
// then invokes opts.ReplaceAttr if set.
//
// Example:
//
//	opts := &slog.HandlerOptions{
//		Level: LevelTrace,
//		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
//			if a.Key == "drop" {
//				return slog.Attr{}
//			}
//			return a
//		},
//	}
//	logger := slog.New(slog.NewJSONHandler(w, MergeWithCustomLevels(opts)))
//	logger.Log(context.Background(), LevelTrace, "message", "hello", "world", "drop", "dropped")
//	// Output example: {"level":"TRACE","msg":"message","hello":"world"}
func MergeWithCustomLevels(opts *slog.HandlerOptions) *slog.HandlerOptions {
	var merged slog.HandlerOptions
	if opts != nil {
		merged = *opts
	}

	next := merged.ReplaceAttr
	merged.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
		a = replaceLevelKey(a)

		if next != nil {
			return next(groups, a)
		}

		return a
	}

	return &merged
}

func replaceLevelKey(a slog.Attr) slog.Attr {
	if a.Key != slog.LevelKey {
		return a
	}

	lvl, ok := a.Value.Any().(slog.Level)
	if !ok {
		return a
	}

	switch lvl {
	case LevelTrace:
		a.Value = slog.StringValue(LevelTraceKey)
	case LevelFatal:
		a.Value = slog.StringValue(LevelFatalKey)
	case LevelPanic:
		a.Value = slog.StringValue(LevelPanicKey)
	default:
		return a
	}

	return a
}

// WithGlobalLevel returns a copy of opts (or a new [slog.HandlerOptions] if opts is nil)
// with the global [slog.LevelVar] set as the level.
func WithGlobalLevel(opts *slog.HandlerOptions) *slog.HandlerOptions {
	var merged slog.HandlerOptions
	if opts != nil {
		merged = *opts
	}
	merged.Level = globalLevel
	return &merged
}

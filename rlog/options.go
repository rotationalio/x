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

	// If ReplaceAttr is nil, set it to ReplaceLevelKey, otherwise chain it.
	if merged.ReplaceAttr == nil {
		merged.ReplaceAttr = ReplaceLevelKey
	} else {
		next := merged.ReplaceAttr
		merged.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
			a = ReplaceLevelKey(groups, a)
			return next(groups, a)
		}
	}

	return &merged
}

// ReplaceLevelKey replaces the level key with the custom level key. Use this
// as a ReplaceAttr function for a [slog.HandlerOptions]. If you wish to merge
// this with other ReplaceAttr functions, you can use [MergeWithCustomLevels].
//
// Example:
//
//	opts := &slog.HandlerOptions{
//		ReplaceAttr: rlog.ReplaceLevelKey,
//	}
//	logger := slog.New(slog.NewJSONHandler(w, opts))
//	logger.Log(context.Background(), rlog.LevelTrace, "message", "hello", "world")
func ReplaceLevelKey(_ []string, a slog.Attr) slog.Attr {
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

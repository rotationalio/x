package rlog

import "log/slog"

// Custom severity levels. Larger values mean more severe events; more negative
// means more verbose.
const (
	// LevelTrace is more verbose than [slog.LevelDebug].
	LevelTrace    slog.Level = slog.LevelDebug - 4
	LevelTraceKey string     = "TRACE"

	// LevelFatal is more severe than [slog.LevelError].
	LevelFatal    slog.Level = slog.LevelError + 4
	LevelFatalKey string     = "FATAL"

	// LevelPanic is more severe than LevelFatal.
	LevelPanic    slog.Level = LevelFatal + 4
	LevelPanicKey string     = "PANIC"
)

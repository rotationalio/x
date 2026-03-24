package rlog

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

// Custom severity levels. Larger values mean more severe events; more negative
// means more verbose.
const (
	// LevelTrace is more verbose than [slog.LevelDebug].
	LevelTrace    slog.Level = slog.LevelDebug - 4
	LevelTraceKey string     = "TRACE"

	// [slog.Level] keys (unavailable from slog package)
	LevelDebugKey string = "DEBUG" // for [slog.LevelDebug]
	LevelInfoKey  string = "INFO"  // for [slog.LevelInfo]
	LevelWarnKey  string = "WARN"  // for [slog.LevelWarn]
	LevelErrorKey string = "ERROR" // for [slog.LevelError]

	// LevelFatal is more severe than [slog.LevelError].
	LevelFatal    slog.Level = slog.LevelError + 4
	LevelFatalKey string     = "FATAL"

	// LevelPanic is more severe than LevelFatal.
	LevelPanic    slog.Level = LevelFatal + 4
	LevelPanicKey string     = "PANIC"
)

// =============================================================================
// LevelDecoder (confire decoder interface)
// =============================================================================

// LevelDecoder deserializes the log level from a config string.
type LevelDecoder slog.Level

// Decode implements confire Decoder interface.
func (ll *LevelDecoder) Decode(value string) error {
	value = strings.TrimSpace(strings.ToUpper(value))
	switch value {
	case LevelTraceKey:
		*ll = LevelDecoder(LevelTrace)
	case LevelDebugKey:
		*ll = LevelDecoder(slog.LevelDebug)
	case LevelInfoKey:
		*ll = LevelDecoder(slog.LevelInfo)
	case LevelWarnKey:
		*ll = LevelDecoder(slog.LevelWarn)
	case LevelErrorKey:
		*ll = LevelDecoder(slog.LevelError)
	case LevelFatalKey:
		*ll = LevelDecoder(LevelFatal)
	case LevelPanicKey:
		*ll = LevelDecoder(LevelPanic)
	default:
		return fmt.Errorf("unknown log level %q", value)
	}
	return nil
}

// Level returns the slog.Level for use when setting handler level.
func (ll LevelDecoder) Level() slog.Level {
	return slog.Level(ll)
}

// Encode converts the log level into a string for use in YAML and JSON.
func (ll LevelDecoder) Encode() (string, error) {
	switch slog.Level(ll) {
	case LevelTrace:
		return LevelTraceKey, nil
	case slog.LevelDebug:
		return LevelDebugKey, nil
	case slog.LevelInfo:
		return LevelInfoKey, nil
	case slog.LevelWarn:
		return LevelWarnKey, nil
	case slog.LevelError:
		return LevelErrorKey, nil
	case LevelFatal:
		return LevelFatalKey, nil
	case LevelPanic:
		return LevelPanicKey, nil
	default:
		return "", fmt.Errorf("unknown log level %d", ll)
	}
}

func (ll LevelDecoder) String() string {
	ls, _ := ll.Encode()
	return ls
}

// UnmarshalJSON implements json.Unmarshaler
func (ll *LevelDecoder) UnmarshalJSON(data []byte) error {
	var ls string
	if err := json.Unmarshal(data, &ls); err != nil {
		return err
	}
	return ll.Decode(ls)
}

// MarshalJSON implements json.Marshaler
func (ll LevelDecoder) MarshalJSON() ([]byte, error) {
	ls, err := ll.Encode()
	if err != nil {
		return nil, err
	}
	return json.Marshal(ls)
}

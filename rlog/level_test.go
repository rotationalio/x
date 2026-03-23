package rlog_test

import (
	"encoding/json"
	"log/slog"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/rlog"
)

func TestLevelDecoder_Decode(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{rlog.LevelTraceKey, rlog.LevelTrace},
		{rlog.LevelDebugKey, slog.LevelDebug},
		{rlog.LevelInfoKey, slog.LevelInfo},
		{rlog.LevelWarnKey, slog.LevelWarn},
		{rlog.LevelErrorKey, slog.LevelError},
		{rlog.LevelFatalKey, rlog.LevelFatal},
		{rlog.LevelPanicKey, rlog.LevelPanic},
		{"  warn ", slog.LevelWarn},
		{"WaRn", slog.LevelWarn},
	}
	for _, tc := range tests {
		var ld rlog.LevelDecoder
		assert.Ok(t, ld.Decode(tc.input))
		assert.Equal(t, tc.want, ld.Level())
	}
}

func TestLevelDecoder_Decode_unknown(t *testing.T) {
	var ld rlog.LevelDecoder
	err := ld.Decode("nope")
	assert.EqualError(t, err, `unknown log level "NOPE"`)
}

func TestLevelDecoder_JSON(t *testing.T) {
	tests := []struct {
		key  string
		want slog.Level
	}{
		{rlog.LevelTraceKey, rlog.LevelTrace},
		{rlog.LevelDebugKey, slog.LevelDebug},
		{rlog.LevelInfoKey, slog.LevelInfo},
		{rlog.LevelWarnKey, slog.LevelWarn},
		{rlog.LevelErrorKey, slog.LevelError},
		{rlog.LevelFatalKey, rlog.LevelFatal},
		{rlog.LevelPanicKey, rlog.LevelPanic},
	}
	for _, tc := range tests {
		var enc rlog.LevelDecoder
		assert.Ok(t, enc.Decode(tc.key))

		b, err := json.Marshal(enc)
		assert.Ok(t, err)
		assert.Equal(t, `"`+tc.key+`"`, string(b))

		var got rlog.LevelDecoder
		assert.Ok(t, json.Unmarshal(b, &got))
		assert.Equal(t, tc.want, got.Level())
	}
}

func TestLevelDecoder_Encode_unknown(t *testing.T) {
	ld := rlog.LevelDecoder(999)
	_, err := ld.Encode()
	assert.EqualError(t, err, "unknown log level 999")
}

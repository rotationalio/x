package console_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"testing/synctest"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/rlog"
	"go.rtnl.ai/x/rlog/console"
)

func TestHandler(t *testing.T) {
	testCases := []struct {
		name string
		opts *console.Options
		want []byte
	}{
		{
			name: "Default",
			opts: nil,
			want: []byte("\x1b[37m[18:00:00.000]\x1b[0m \x1b[92mINFO:\x1b[0m \x1b[97minfo\x1b[0m {}\n\x1b[37m[18:00:00.000]\x1b[0m \x1b[93mWARN:\x1b[0m \x1b[97mwarning\x1b[0m {}\n\x1b[37m[18:00:00.000]\x1b[0m \x1b[91mERROR:\x1b[0m \x1b[97merror\x1b[0m {}\n"),
		},
		{
			name: "NoColor",
			opts: &console.Options{NoColor: true},
			want: []byte("[18:00:00.000] INFO: info {}\n[18:00:00.000] WARN: warning {}\n[18:00:00.000] ERROR: error {}\n"),
		},
		{
			name: "NoJSON",
			opts: &console.Options{NoJSON: true},
			want: []byte("\x1b[37m[18:00:00.000]\x1b[0m \x1b[92mINFO:\x1b[0m \x1b[97minfo\x1b[0m\n\x1b[37m[18:00:00.000]\x1b[0m \x1b[93mWARN:\x1b[0m \x1b[97mwarning\x1b[0m\n\x1b[37m[18:00:00.000]\x1b[0m \x1b[91mERROR:\x1b[0m \x1b[97merror\x1b[0m\n"),
		},
		{
			name: "Plain",
			opts: &console.Options{NoColor: true, NoJSON: true},
			want: []byte("[18:00:00.000] INFO: info\n[18:00:00.000] WARN: warning\n[18:00:00.000] ERROR: error\n"),
		},
		{
			name: "IndentJSON",
			opts: &console.Options{IndentJSON: true},
			want: []byte("\x1b[37m[18:00:00.000]\x1b[0m \x1b[92mINFO:\x1b[0m \x1b[97minfo\x1b[0m {}\n\x1b[37m[18:00:00.000]\x1b[0m \x1b[93mWARN:\x1b[0m \x1b[97mwarning\x1b[0m {}\n\x1b[37m[18:00:00.000]\x1b[0m \x1b[91mERROR:\x1b[0m \x1b[97merror\x1b[0m {}\n"),
		},
		{
			name: "Trace",
			opts: &console.Options{HandlerOptions: &slog.HandlerOptions{Level: rlog.LevelTrace}},
			want: []byte("\x1b[37m[18:00:00.000]\x1b[0m \x1b[37mDEBUG-4:\x1b[0m \x1b[97mtracing\x1b[0m {}\n\x1b[37m[18:00:00.000]\x1b[0m \x1b[36mDEBUG:\x1b[0m \x1b[97mdebugging\x1b[0m {}\n\x1b[37m[18:00:00.000]\x1b[0m \x1b[92mINFO:\x1b[0m \x1b[97minfo\x1b[0m {}\n\x1b[37m[18:00:00.000]\x1b[0m \x1b[93mWARN:\x1b[0m \x1b[97mwarning\x1b[0m {}\n\x1b[37m[18:00:00.000]\x1b[0m \x1b[91mERROR:\x1b[0m \x1b[97merror\x1b[0m {}\n"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var buf bytes.Buffer
				logger := rlog.New(slog.New(console.New(&buf, tc.opts)))

				logger.Trace("tracing")
				logger.Debug("debugging")
				logger.Info("info")
				logger.Warn("warning")
				logger.Error("error")

				assert.True(t, bytes.Equal(buf.Bytes(), tc.want))
			})
		})
	}
}

func TestHandler_WithAttrs(t *testing.T) {
	testCases := []struct {
		name string
		opts *console.Options
		want []byte
	}{
		{
			name: "Default",
			opts: nil,
			want: []byte("\x1b[37m[18:00:00.000]\x1b[0m \x1b[92mINFO:\x1b[0m \x1b[97minfo\x1b[0m {\"foo\":\"bar\",\"user\":\"alice\"}\n\x1b[37m[18:00:00.000]\x1b[0m \x1b[93mWARN:\x1b[0m \x1b[97mwarning\x1b[0m {\"code\":404,\"error\":\"not found\"}\n\x1b[37m[18:00:00.000]\x1b[0m \x1b[91mERROR:\x1b[0m \x1b[97merror\x1b[0m {\"code\":400,\"error\":\"uncapped jar\"}\n"),
		},
		{
			name: "NoColor",
			opts: &console.Options{NoColor: true},
			want: []byte("[18:00:00.000] INFO: info {\"foo\":\"bar\",\"user\":\"alice\"}\n[18:00:00.000] WARN: warning {\"code\":404,\"error\":\"not found\"}\n[18:00:00.000] ERROR: error {\"code\":400,\"error\":\"uncapped jar\"}\n"),
		},
		{
			name: "NoJSON",
			opts: &console.Options{NoJSON: true},
			want: []byte("\x1b[37m[18:00:00.000]\x1b[0m \x1b[92mINFO:\x1b[0m \x1b[97minfo\x1b[0m\n\x1b[37m[18:00:00.000]\x1b[0m \x1b[93mWARN:\x1b[0m \x1b[97mwarning\x1b[0m\n\x1b[37m[18:00:00.000]\x1b[0m \x1b[91mERROR:\x1b[0m \x1b[97merror\x1b[0m\n"),
		},
		{
			name: "Plain",
			opts: &console.Options{NoColor: true, NoJSON: true},
			want: []byte("[18:00:00.000] INFO: info\n[18:00:00.000] WARN: warning\n[18:00:00.000] ERROR: error\n"),
		},
		{
			name: "IndentJSON",
			opts: &console.Options{IndentJSON: true},
			want: []byte("\x1b[37m[18:00:00.000]\x1b[0m \x1b[92mINFO:\x1b[0m \x1b[97minfo\x1b[0m {\n  \"foo\": \"bar\",\n  \"user\": \"alice\"\n}\n\x1b[37m[18:00:00.000]\x1b[0m \x1b[93mWARN:\x1b[0m \x1b[97mwarning\x1b[0m {\n  \"code\": 404,\n  \"error\": \"not found\"\n}\n\x1b[37m[18:00:00.000]\x1b[0m \x1b[91mERROR:\x1b[0m \x1b[97merror\x1b[0m {\n  \"code\": 400,\n  \"error\": \"uncapped jar\"\n}\n"),
		},
		{
			name: "IndentJSONNoColor",
			opts: &console.Options{IndentJSON: true, NoColor: true},
			want: []byte("[18:00:00.000] INFO: info {\n  \"foo\": \"bar\",\n  \"user\": \"alice\"\n}\n[18:00:00.000] WARN: warning {\n  \"code\": 404,\n  \"error\": \"not found\"\n}\n[18:00:00.000] ERROR: error {\n  \"code\": 400,\n  \"error\": \"uncapped jar\"\n}\n"),
		},
		{
			name: "Trace",
			opts: &console.Options{HandlerOptions: &slog.HandlerOptions{Level: rlog.LevelTrace}},
			want: []byte("\x1b[37m[18:00:00.000]\x1b[0m \x1b[37mDEBUG-4:\x1b[0m \x1b[97mtracing\x1b[0m {\"foo\":\"bar\"}\n\x1b[37m[18:00:00.000]\x1b[0m \x1b[36mDEBUG:\x1b[0m \x1b[97mdebugging\x1b[0m {\"foo\":\"bar\",\"user\":\"bob\"}\n\x1b[37m[18:00:00.000]\x1b[0m \x1b[92mINFO:\x1b[0m \x1b[97minfo\x1b[0m {\"foo\":\"bar\",\"user\":\"alice\"}\n\x1b[37m[18:00:00.000]\x1b[0m \x1b[93mWARN:\x1b[0m \x1b[97mwarning\x1b[0m {\"code\":404,\"error\":\"not found\"}\n\x1b[37m[18:00:00.000]\x1b[0m \x1b[91mERROR:\x1b[0m \x1b[97merror\x1b[0m {\"code\":400,\"error\":\"uncapped jar\"}\n"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var buf bytes.Buffer
				logger := rlog.New(slog.New(console.New(&buf, tc.opts)))
				ctx := context.Background()

				logger.TraceAttrs(ctx, "tracing", slog.String("foo", "bar"))
				logger.DebugAttrs(ctx, "debugging", slog.String("foo", "bar"), slog.String("user", "bob"))
				logger.InfoAttrs(ctx, "info", slog.String("foo", "bar"), slog.String("user", "alice"))
				logger.WarnAttrs(ctx, "warning", slog.String("error", "not found"), slog.Int("code", 404))
				logger.ErrorAttrs(ctx, "error", slog.String("error", "uncapped jar"), slog.Int("code", 400))

				assert.True(t, bytes.Equal(buf.Bytes(), tc.want))
			})
		})
	}
}

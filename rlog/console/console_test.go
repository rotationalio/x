package console_test

import (
	"bytes"
	"context"
	"log/slog"
	"runtime"
	"strings"
	"testing"
	"testing/slogtest"
	"testing/synctest"
	"time"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/rlog"
	"go.rtnl.ai/x/rlog/console"
)

// TestHandler_GoldenTest tests the [Handler] with various options and verifies
// the output matches the golden bytes. NOTE: [slog.HandlerOptions.AddSource] is
// not tested here because it is hard to verify the output matches the golden bytes;
// we have a separate test for [slog.HandlerOptions.AddSource] called
// [TestHandler_AddSource_prefix].
func TestHandler_GoldenTest(t *testing.T) {
	testCases := []struct {
		name string
		opts *console.Options
		want []byte
	}{
		{
			name: "Default",
			opts: nil,
			want: []byte("\x1b[37m[00:00:00.000]\x1b[0m \x1b[92mINFO:\x1b[0m \x1b[97minfo\x1b[0m {\"req\":{\"foo\":\"bar\",\"id\":\"1\",\"user\":\"alice\"},\"svc\":\"api\"}\n\x1b[37m[00:00:00.000]\x1b[0m \x1b[93mWARN:\x1b[0m \x1b[97mwarning\x1b[0m {\"req\":{\"code\":404,\"error\":\"not found\",\"id\":\"1\"},\"svc\":\"api\"}\n\x1b[37m[00:00:00.000]\x1b[0m \x1b[91mERROR:\x1b[0m \x1b[97merror\x1b[0m {\"req\":{\"code\":400,\"error\":\"uncapped jar\",\"id\":\"1\"},\"svc\":\"api\"}\n"),
		},
		{
			name: "NoColor",
			opts: &console.Options{NoColor: true},
			want: []byte("[00:00:00.000] INFO: info {\"req\":{\"foo\":\"bar\",\"id\":\"1\",\"user\":\"alice\"},\"svc\":\"api\"}\n[00:00:00.000] WARN: warning {\"req\":{\"code\":404,\"error\":\"not found\",\"id\":\"1\"},\"svc\":\"api\"}\n[00:00:00.000] ERROR: error {\"req\":{\"code\":400,\"error\":\"uncapped jar\",\"id\":\"1\"},\"svc\":\"api\"}\n"),
		},
		{
			name: "NoJSON",
			opts: &console.Options{NoJSON: true},
			want: []byte("\x1b[37m[00:00:00.000]\x1b[0m \x1b[92mINFO:\x1b[0m \x1b[97minfo\x1b[0m\n\x1b[37m[00:00:00.000]\x1b[0m \x1b[93mWARN:\x1b[0m \x1b[97mwarning\x1b[0m\n\x1b[37m[00:00:00.000]\x1b[0m \x1b[91mERROR:\x1b[0m \x1b[97merror\x1b[0m\n"),
		},
		{
			name: "Plain",
			opts: &console.Options{NoColor: true, NoJSON: true},
			want: []byte("[00:00:00.000] INFO: info\n[00:00:00.000] WARN: warning\n[00:00:00.000] ERROR: error\n"),
		},
		{
			name: "IndentJSON",
			opts: &console.Options{IndentJSON: true},
			want: []byte("\x1b[37m[00:00:00.000]\x1b[0m \x1b[92mINFO:\x1b[0m \x1b[97minfo\x1b[0m {\n  \"req\": {\n    \"foo\": \"bar\",\n    \"id\": \"1\",\n    \"user\": \"alice\"\n  },\n  \"svc\": \"api\"\n}\n\x1b[37m[00:00:00.000]\x1b[0m \x1b[93mWARN:\x1b[0m \x1b[97mwarning\x1b[0m {\n  \"req\": {\n    \"code\": 404,\n    \"error\": \"not found\",\n    \"id\": \"1\"\n  },\n  \"svc\": \"api\"\n}\n\x1b[37m[00:00:00.000]\x1b[0m \x1b[91mERROR:\x1b[0m \x1b[97merror\x1b[0m {\n  \"req\": {\n    \"code\": 400,\n    \"error\": \"uncapped jar\",\n    \"id\": \"1\"\n  },\n  \"svc\": \"api\"\n}\n"),
		},
		{
			name: "IndentJSONNoColor",
			opts: &console.Options{IndentJSON: true, NoColor: true},
			want: []byte("[00:00:00.000] INFO: info {\n  \"req\": {\n    \"foo\": \"bar\",\n    \"id\": \"1\",\n    \"user\": \"alice\"\n  },\n  \"svc\": \"api\"\n}\n[00:00:00.000] WARN: warning {\n  \"req\": {\n    \"code\": 404,\n    \"error\": \"not found\",\n    \"id\": \"1\"\n  },\n  \"svc\": \"api\"\n}\n[00:00:00.000] ERROR: error {\n  \"req\": {\n    \"code\": 400,\n    \"error\": \"uncapped jar\",\n    \"id\": \"1\"\n  },\n  \"svc\": \"api\"\n}\n"),
		},
		{
			name: "Trace",
			opts: &console.Options{HandlerOptions: &slog.HandlerOptions{Level: rlog.LevelTrace}},
			want: []byte("\x1b[37m[00:00:00.000]\x1b[0m \x1b[37mDEBUG-4:\x1b[0m \x1b[97mtracing\x1b[0m {\"req\":{\"foo\":\"bar\",\"id\":\"1\"},\"svc\":\"api\"}\n\x1b[37m[00:00:00.000]\x1b[0m \x1b[36mDEBUG:\x1b[0m \x1b[97mdebugging\x1b[0m {\"req\":{\"foo\":\"bar\",\"id\":\"1\",\"user\":\"bob\"},\"svc\":\"api\"}\n\x1b[37m[00:00:00.000]\x1b[0m \x1b[92mINFO:\x1b[0m \x1b[97minfo\x1b[0m {\"req\":{\"foo\":\"bar\",\"id\":\"1\",\"user\":\"alice\"},\"svc\":\"api\"}\n\x1b[37m[00:00:00.000]\x1b[0m \x1b[93mWARN:\x1b[0m \x1b[97mwarning\x1b[0m {\"req\":{\"code\":404,\"error\":\"not found\",\"id\":\"1\"},\"svc\":\"api\"}\n\x1b[37m[00:00:00.000]\x1b[0m \x1b[91mERROR:\x1b[0m \x1b[97merror\x1b[0m {\"req\":{\"code\":400,\"error\":\"uncapped jar\",\"id\":\"1\"},\"svc\":\"api\"}\n"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var buf bytes.Buffer
				opts := tc.opts
				if opts == nil {
					opts = &console.Options{}
				}
				opts.UTCTime = true
				base := rlog.New(slog.New(console.New(&buf, opts)))
				log := base.With(slog.String("svc", "api")).WithGroup("req").With(slog.String("id", "1"))
				ctx := context.Background()

				log.TraceAttrs(ctx, "tracing", slog.String("foo", "bar"))
				log.DebugAttrs(ctx, "debugging", slog.String("foo", "bar"), slog.String("user", "bob"))
				log.InfoAttrs(ctx, "info", slog.String("foo", "bar"), slog.String("user", "alice"))
				log.WarnAttrs(ctx, "warning", slog.String("error", "not found"), slog.Int("code", 404))
				log.ErrorAttrs(ctx, "error", slog.String("error", "uncapped jar"), slog.Int("code", 400))

				assert.True(t, bytes.Equal(buf.Bytes(), tc.want), "got:\n%q \nwant:\n%q", buf.Bytes(), tc.want)
			})
		})
	}
}

// TestHandler_slogtest runs [testing/slogtest.TestHandler] against [console.Handler]; each emitted line
// is parsed with [console.ParseLogLine] to build the result maps. Options use plain text and compact JSON
// so one line equals one log record.
func TestHandler_slogtest(t *testing.T) {
	var buf bytes.Buffer
	opts := &console.Options{
		HandlerOptions: &slog.HandlerOptions{Level: slog.LevelInfo},
		NoColor:        true,
		UTCTime:        true,
	}
	h := console.New(&buf, opts)
	err := slogtest.TestHandler(h, func() []map[string]any {
		var maps []map[string]any
		for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
			if line == "" {
				continue
			}
			m, e := console.ParseLogLine(line)
			if e != nil {
				t.Fatalf("ParseLogLine(%q): %v", line, e)
			}
			maps = append(maps, m)
		}
		return maps
	})
	assert.Ok(t, err, "handler should satisfy slogtest when each line is round-tripped through ParseLogLine")
}

// TestHandler_WithGroup_mergedIntoJSON checks that logger.With("a","b").WithGroup("G") nests record attrs
// under G in the JSON tail and that [console.ParseLogLine] sees the same shape.
func TestHandler_WithGroup_mergedIntoJSON(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var buf bytes.Buffer
		opts := &console.Options{NoColor: true, UTCTime: true}
		log := slog.New(console.New(&buf, opts)).With("a", "b").WithGroup("G")
		log.Info("msg", "k", "v")

		m, err := console.ParseLogLine(strings.TrimSpace(buf.String()))
		assert.Ok(t, err, "single Info line with group should parse")
		g, ok := m["G"].(map[string]any)
		assert.True(t, ok, `top-level key "G" should be a nested map from WithGroup`)
		assert.Equal(t, "v", g["k"], "record attr k should appear inside group G")
		assert.Equal(t, "msg", m[slog.MessageKey], "msg should be the log message text")
	})
}

// TestHandler_ReplaceAttr_dropsKey verifies [slog.HandlerOptions.ReplaceAttr] removes a user key from
// the JSON object while leaving other attrs (JSON numbers decode as float64 in the parsed map).
func TestHandler_ReplaceAttr_dropsKey(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var buf bytes.Buffer
		opts := &console.Options{
			NoColor: true,
			UTCTime: true,
			HandlerOptions: &slog.HandlerOptions{
				Level: slog.LevelInfo,
				ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
					if a.Key == "drop" {
						return slog.Attr{}
					}
					return a
				},
			},
		}
		log := slog.New(console.New(&buf, opts))
		log.Info("x", "keep", 1, "drop", "gone")

		m, err := console.ParseLogLine(strings.TrimSpace(buf.String()))
		assert.Ok(t, err, "line with ReplaceAttr should still parse")
		_, hasDrop := m["drop"]
		assert.False(t, hasDrop, "ReplaceAttr should strip key drop from output")
		assert.Equal(t, float64(1), m["keep"], "keep should remain and match JSON-decoded int")
	})
}

// TestHandler_AddSource_prefix ensures AddSource adds a basename:line prefix and ParseLogLine exposes
// [slog.SourceKey] with File and Line when Handle receives a record built with a non-zero PC.
func TestHandler_AddSource_prefix(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var buf bytes.Buffer
		opts := &console.Options{
			NoColor: true,
			UTCTime: true,
			HandlerOptions: &slog.HandlerOptions{
				Level:     slog.LevelInfo,
				AddSource: true,
			},
		}
		h := console.New(&buf, opts)
		pc := callerPC(2)
		r := slog.NewRecord(time.Now(), slog.LevelInfo, "hello", pc)
		assert.Ok(t, h.Handle(context.Background(), r), "Handle with AddSource should not error")

		line := strings.TrimSpace(buf.String())
		assert.True(t, strings.Contains(line, ":"), "AddSource should print file:line before the rest of the line; got %q", line)

		m, err := console.ParseLogLine(line)
		assert.Ok(t, err, "line with source prefix should parse")
		src, ok := m[slog.SourceKey].(map[string]any)
		assert.True(t, ok, "SourceKey should be a map[string]any like JSON source objects")
		_, hasFile := src["File"]
		assert.True(t, hasFile, "parsed source should include File")
		_, hasLine := src["Line"]
		assert.True(t, hasLine, "parsed source should include Line")
	})
}

// callerPC returns the PC at the given skip depth; skip=2 yields the test frame so [slog.Record] carries a real line.
func callerPC(skip int) uintptr {
	var pcs [1]uintptr
	runtime.Callers(skip, pcs[:])
	return pcs[0]
}

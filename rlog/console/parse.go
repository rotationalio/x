package console

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Line layout: optional source, optional [HH:MM:SS.mmm], LEVEL: message [JSON].
var (
	ansiRegexp     = regexp.MustCompile(`\x1b\[[0-9;]*m`)
	sourcePrefixRx = regexp.MustCompile(`^([^\s]+:\d+)\s+`)
	timeBracketRx  = regexp.MustCompile(`^\[(\d{2}:\d{2}:\d{2}\.\d{3})\]\s+`)
	levelPrefixRx  = regexp.MustCompile(`^([A-Za-z0-9+-]+):\s+`)
)

// ParseLogLine parses a single line from [Handler] into a map, e.g. for [testing/slogtest.TestHandler].
// Expects one line per record (no IndentJSON); strips ANSI escapes.
func ParseLogLine(line string) (map[string]any, error) {
	s := strings.TrimSpace(stripAnsi(line))
	if s == "" {
		return nil, fmt.Errorf("empty line")
	}

	m := make(map[string]any)

	// Parse optional source:line.
	if sm := sourcePrefixRx.FindStringSubmatch(s); sm != nil {
		file, lineNum, ok := parseSourceToken(sm[1])
		if ok {
			m[slog.SourceKey] = map[string]any{
				"File": file,
				"Line": float64(lineNum),
			}
		}
		s = strings.TrimSpace(s[len(sm[0]):])
	}

	// Parse optional time bracket.
	if tm := timeBracketRx.FindStringSubmatch(s); tm != nil {
		clock, err := time.ParseInLocation("15:04:05.000", tm[1], time.UTC)
		if err != nil {
			return nil, fmt.Errorf("time bracket: %w", err)
		}
		t := time.Date(2000, 1, 1, clock.Hour(), clock.Minute(), clock.Second(), clock.Nanosecond(), time.UTC)
		m[slog.TimeKey] = t
		s = strings.TrimSpace(s[len(tm[0]):])
	}

	// Parse level.
	lm := levelPrefixRx.FindStringSubmatch(s)
	if lm == nil {
		return nil, fmt.Errorf("no level prefix in %q", s)
	}
	levelStr := lm[1]
	m[slog.LevelKey] = levelStr
	s = strings.TrimSpace(s[len(lm[0]):])

	// Parse JSON tail.
	jsonPart, jStart, ok := extractOuterJSONObject(s)
	if ok {
		// Parse message.
		m[slog.MessageKey] = strings.TrimSpace(s[:jStart])

		// Parse JSON.
		var obj map[string]any
		if err := json.Unmarshal([]byte(jsonPart), &obj); err != nil {
			return nil, fmt.Errorf("json tail: %w", err)
		}
		maps.Copy(m, obj)
	} else {
		// Parse message.
		m[slog.MessageKey] = strings.TrimSpace(s)
	}

	return m, nil
}

// stripAnsi removes SGR sequences from s.
func stripAnsi(s string) string {
	return ansiRegexp.ReplaceAllString(s, "")
}

// extractOuterJSONObject finds the last top-level {...} in s (handles nested braces).
func extractOuterJSONObject(s string) (json string, start int, ok bool) {
	end := strings.LastIndex(s, "}")
	if end < 0 {
		return "", 0, false
	}
	depth := 0
	for i := end; i >= 0; i-- {
		switch s[i] {
		case '}':
			depth++
		case '{':
			depth--
			if depth == 0 {
				return strings.TrimSpace(s[i : end+1]), i, true
			}
		}
	}
	return "", 0, false
}

// parseSourceToken splits "file:line" (basename may contain ':' on some platforms — last ':' wins).
func parseSourceToken(tok string) (file string, line int, ok bool) {
	i := strings.LastIndex(tok, ":")
	if i <= 0 || i == len(tok)-1 {
		return "", 0, false
	}
	file = tok[:i]
	n, err := strconv.Atoi(tok[i+1:])
	if err != nil {
		return "", 0, false
	}
	return file, n, true
}

package dsn

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const ReadOnly = "readonly"

// Additional options for establishing a database connection.
// Option keys are case-insensitive.
type Options map[string]string

// Returns true iff the readonly option is set and is true.
func (o Options) ReadOnly() bool {
	v, ok := o.Get(ReadOnly)
	if !ok {
		return false
	}

	if ro, err := strconv.ParseBool(v); err == nil {
		return ro
	}
	return false
}

func (o Options) Set(key string, value any) {
	switch v := value.(type) {
	case string:
		o[key] = v
	case bool:
		o[key] = strconv.FormatBool(v)
	case int:
		o[key] = strconv.Itoa(v)
	case int64:
		o[key] = strconv.FormatInt(v, 10)
	case float64:
		o[key] = strconv.FormatFloat(v, 'f', -1, 64)
	case time.Duration:
		o[key] = v.String()
	case fmt.Stringer:
		o[key] = v.String()
	default:
		o[key] = fmt.Sprintf("%v", v)
	}
}

// Attempt to get the value of an option key using several forms of the key.
// First the exact key is tried, then the key in lowercase and uppercase.
func (o Options) Get(key string) (v string, ok bool) {
	// Try original value.
	if v, ok = o[key]; ok {
		return v, ok
	}

	// Try lowercase value.
	if v, ok = o[strings.ToLower(key)]; ok {
		return v, ok
	}

	// Try uppercase value.
	if v, ok = o[strings.ToUpper(key)]; ok {
		return v, ok
	}

	return "", false
}

func (o Options) Bool(key string) (v bool, err error) {
	var (
		s  string
		ok bool
	)

	if s, ok = o.Get(key); !ok {
		return false, fmt.Errorf("option %q not found", key)
	}

	if v, err = strconv.ParseBool(s); err != nil {
		return false, fmt.Errorf("could not parse %q as bool: %w", key, err)
	}
	return v, nil
}

func (o Options) Int64(key string) (v int64, err error) {
	var (
		s  string
		ok bool
	)

	if s, ok = o.Get(key); !ok {
		return 0, fmt.Errorf("option %q not found", key)
	}

	if v, err = strconv.ParseInt(s, 10, 64); err != nil {
		return 0, fmt.Errorf("could not parse %q as int64: %w", key, err)
	}
	return v, nil
}

func (o Options) Float64(key string) (v float64, err error) {
	var (
		s  string
		ok bool
	)

	if s, ok = o.Get(key); !ok {
		return 0, fmt.Errorf("option %q not found", key)
	}

	if v, err = strconv.ParseFloat(s, 64); err != nil {
		return 0, fmt.Errorf("could not parse %q as float64: %w", key, err)
	}
	return v, nil
}

func (o Options) Duration(key string) (v time.Duration, err error) {
	var (
		s  string
		ok bool
	)

	if s, ok = o.Get(key); !ok {
		return 0, fmt.Errorf("option %q not found", key)
	}

	if v, err = time.ParseDuration(s); err != nil {
		return 0, fmt.Errorf("could not parse %q as duration: %w", key, err)
	}
	return v, nil
}

package console

import "log/slog"

// Options are used for the [console.Handler] a nil or zero valued Options consists
// entirely of default values.
type Options struct {
	// The [slog.HandlerOptions] to use for the [console.Handler].
	*slog.HandlerOptions

	// Marking true causes the handler to format the logs without terminal colors.
	NoColor bool

	// Marking true causes the handler to indent the JSON dictionary on multiple lines.
	IndentJSON bool

	// Marking true causes the handler to format the logs without JSON dictionary of attributes.
	NoJSON bool

	// Marking true causes the handler to format the time using the UTC timezone.
	UTCTime bool
}

package console

import (
	"log/slog"

	"go.rtnl.ai/x/rlog"
)

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

// MergeWithCustomLevels merges the options with the custom rlog.Level* keys.
// If o is nil, returns a new Options with the default values.
func (o *Options) MergeWithCustomLevels() *Options {
	if o == nil {
		o = &Options{}
	}
	return &Options{
		HandlerOptions: rlog.MergeWithCustomLevels(o.HandlerOptions),
		NoColor:        o.NoColor,
		IndentJSON:     o.IndentJSON,
		NoJSON:         o.NoJSON,
		UTCTime:        o.UTCTime,
	}
}

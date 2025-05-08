package typecase

import (
	"strings"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	initialisms = []string{
		// Borrowed from Staticcheck (https://github.com/dominikh/go-tools/blob/master/config/config.go#L167)
		"ACL", "API", "ASCII", "CPU", "CSS", "DNS",
		"EOF", "GUID", "HTML", "HTTP", "HTTPS", "ID",
		"IP", "JSON", "QPS", "RAM", "RPC", "SLA",
		"SMTP", "SQL", "SSH", "TCP", "TLS", "TTL",
		"UDP", "UI", "GID", "UID", "UUID", "URI",
		"URL", "UTF8", "VM", "XML", "XMPP", "XSRF",
		"XSS", "SIP", "RTP", "AMQP", "DB", "TS",

		// Media initialisms
		"1080P", "2D", "3D", "4K", "8K", "AAC", "AC3",
		"CDN", "DASH", "DRM", "DVR", "EAC3", "FPS", "GOP",
		"H264", "H265", "HD", "HLS", "MJPEG", "MP2T", "MP3",
		"MP4", "MPEG2", "MPEG4", "NTSC", "PCM", "RGB", "RGBA",
		"RTMP", "RTP", "SCTE", "SCTE35", "SMPTE", "UPID", "UPIDS",
		"VOD", "YUV420", "YUV422", "YUV444",

		// Rotational Custom
		"VASP", "AI", "LLM", "ULID",
	}

	suffixes = []string{
		"D",    // E.g. 2D, 3D
		"GB",   // E.g. 100GB
		"K",    // E.g. 4K, 8K
		"KB",   // E.g. 100KB
		"KBPS", // E.g. 64kbps
		"MB",   // E.g. 100MB
		"MPBS", // E.g. 2500mbps
		"P",    // E.g. 1080P
		"TB",   // E.g. 100TB
	}

	// The lists above are maintained for ease of maintenance but are converted to
	// sets for faster lookups at runtime.
	initialismMap map[string]struct{}
	suffixMap     map[string]struct{}
)

func init() {
	initialismMap = make(map[string]struct{}, len(initialisms))
	for _, initialism := range initialisms {
		initialismMap[initialism] = struct{}{}
	}

	suffixMap = make(map[string]struct{}, len(suffixes))
	for _, suffix := range suffixes {
		suffixMap[suffix] = struct{}{}
	}

	suffixes = nil
	initialisms = nil
}

//===========================================================================
// Primary Case Functions
//===========================================================================

func Camel(s string, formatters ...Formatter) string {
	if formatters == nil {
		formatters = []Formatter{strings.ToLower}
	}

	// TODO: add support for other languages.
	title := cases.Title(language.English)

	f := append(formatters, title.String, Initialism)
	return Join(Split(s), "", f...)
}

func LowerCamel(s string, formatters ...Formatter) string {
	if formatters == nil {
		formatters = []Formatter{strings.ToLower}
	}

	// TODO: add support for other languages.
	title := cases.Title(language.English)
	f := append(formatters, title.String, Initialism)
	parts := Split(s)
	runes := []rune(Join(parts, "", f...))

	if len(parts) > 0 {
		if _, ok := initialismMap[parts[0]]; !ok {
			runes[0] = unicode.ToLower(runes[0])
		} else {
			return strings.Replace(string(runes), parts[0], strings.ToLower(parts[0]), 1)
		}
	}
	return string(runes)
}

func Snake(s string, formatters ...Formatter) string {
	if formatters == nil {
		formatters = []Formatter{strings.ToLower}
	}
	return Join(Split(s), "_", formatters...)
}

func Kebab(s string, formatters ...Formatter) string {
	if formatters == nil {
		formatters = []Formatter{strings.ToLower}
	}
	return Join(Split(s), "-", formatters...)
}

func Constant(s string, formatters ...Formatter) string {
	if formatters == nil {
		formatters = []Formatter{strings.ToUpper}
	}
	return Join(Split(s), "_", formatters...)
}

func Title(s string, formatters ...Formatter) string {
	if formatters == nil {
		formatters = []Formatter{strings.ToLower}
	}

	// TODO: add support for other languages.
	title := cases.Title(language.English)

	f := append(formatters, title.String, Initialism)
	return Join(Split(s), " ", f...)
}

//===========================================================================
// Typecase Utilities
//===========================================================================

// Formatter is used to transform parts of a string to modify how a final case is rendered.
type Formatter func(string) string

// Identity formatter that returns its input unchanged.
func Identity(s string) string {
	return s
}

// Initialism checks if a string is an initialism and keeps it uppercaase.
func Initialism(s string) string {
	S := strings.ToUpper(s)
	if _, ok := initialismMap[S]; ok {
		return S
	}
	return s
}

// Split a value into parts, taking into account different casing styles.
func Split(s string) (out []string) {
	b := 0
	state := stateNone

splitter:
	for i, c := range s {
		// Regardless of state, spaces and punctuation break works.
		// This also handles kabob and snake casing.
		if unicode.IsSpace(c) || unicode.IsPunct(c) {
			if i-b > 0 {
				out = append(out, s[b:i])
			}

			b = i + 1
			state = stateNone
			continue splitter
		}

		switch {
		case state != stateFirstUpper && state != stateUpper && unicode.IsUpper(c):
			// Initial uppercase, might start a word, e.g. for CamelCase.
			if b != i {
				out = append(out, s[b:i])
				b = i
			}
			state = stateFirstUpper
		case state == stateFirstUpper && unicode.IsUpper(c):
			// Group uppercase in case of initialisms or constant casing.
			state = stateUpper
		case state != stateSymbol && !unicode.IsLetter(c):
			// Anything -> non-letter
			if b != i {
				out = append(out, s[b:i])
				b = i
			}
			state = stateSymbol
		case state != stateLower && unicode.IsLower(c):
			// Multi-character uppercase -> lowercase and this is the last time the
			// upercase is part of the string, e.g. JSONMessage.
			if state == stateUpper {
				if i > 0 && b != i-1 {
					out = append(out, s[b:i-1])
					b = i - 1
				}
			} else if state != stateFirstUpper {
				// End of a non-uppercase or non-lowercase string. Ignore the first
				// upper state as its part of the same word.
				if i > 0 && b != i {
					out = append(out, s[b:i])
					b = i
				}
			}
			state = stateLower
		}
	}

	// Include whatever is at the end of the string.
	if b < len(s) {
		out = append(out, s[b:])
	}

	return out
}

// Join the parts of a string together with the separator applying formatters to each
// part in order to get a specifically cased/formatted string. This method also removes
// any empty parts from the final string.
func Join(parts []string, sep string, formatters ...Formatter) string {
	for i := 0; i < len(parts); i++ {
	fmtloop:
		for _, formatter := range formatters {
			parts[i] = formatter(parts[i])
			if parts[i] == "" {
				break fmtloop
			}
		}

		// If the part is empty, remove it from the slice.
		if parts[i] == "" {
			parts = append(parts[:i], parts[i+1:]...)
			i--
		}
	}

	return strings.Join(parts, sep)
}

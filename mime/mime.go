package mime

import (
	"errors"
	"mime"
	"net/http"
	"regexp"
	"strings"
)

var (
	ErrEmptyMimeType   = errors.New("empty mime type")
	ErrInvalidMimeType = errors.New("invalid mime type")
)

// Type is a media type value without optional parameters, per RFC 1521.
// Replaces MIME types with standard strings.
type Type string

// ParseMimeType checks that a type string is a valid media type without any optional
// parameters and converts it into a media.Type value. It does not check to see if the
// mime type is a known type (e.g. a constant in this package).
func ParseMimeType(s string) (out Type, err error) {
	out = Type(s)
	if err = out.Validate(); err != nil {
		return UnknownMimeType, err
	}
	return out, nil
}

// ParseMediaType parses a media type value and any optional parameters, per RFC 1521.
// Media types are the values in Content-Type and Content-Disposition headers (RFC 2183).
// On success, ParseMediaType returns the media type converted to lowercase and trimmed
// of white space and a non-nil map. If there is an error parsing the optional parameter,
// the media type will be returned along with the error ErrInvalidMediaParameter. The
// returned map, params, maps from the lowercase attribute to the attribute value with
// its case preserved.
//
// NOTE: this is a typed wrapper for the mime.ParseMediaType standard library function.
func Parse(s string) (out MimeType, err error) {
	var mediaType string
	if mediaType, out.Params, err = mime.ParseMediaType(s); err != nil {
		return out, err
	}

	out.Type = Type(mediaType)
	if err = out.Type.Validate(); err != nil {
		return out, err
	}
	return out, nil
}

// MimeType represents a media type value and any optional parameters, per RFC 1521.
// Media types are the values in Content-Type and Content-Disposition headers (RFC 2183).
type MimeType struct {
	Type   Type              // the type of the media type including subtype
	Params map[string]string // the optional parameters of the media type from a Content-Type header
}

var (
	mediaTypeRegex = regexp.MustCompile(`^([a-zA-Z0-9!#$&+\-\^_.]+)/([a-zA-Z0-9!#$&+\-\^_.]+)$`)
)

// Validate the media type. Invalid media types do not have both a type and subtype or
// have invalid characters according to RFC 2046. Empty media types are invalid.
// NOTE: this method does not match the constants described in this package.
func (m Type) Validate() (err error) {
	if m == "" {
		return ErrEmptyMimeType
	}
	if !mediaTypeRegex.MatchString(string(m)) {
		return ErrInvalidMimeType
	}
	return nil
}

// Returns the top-level type of the media type; e.g. "text", "image", "audio",
// "video", "application", etc. Used to collect similar media types together.
func (m Type) Top() string {
	groups := mediaTypeRegex.FindStringSubmatch(string(m))
	if len(groups) > 1 {
		return groups[1]
	}
	return ""
}

// String returns the media type as a string.
func (m Type) String() string {
	return string(m)
}

// Returns the subtype of the media type; e.g. "plain", "markdown", "html", "csv", "
// yaml", "json", "xml", "pdf", "jpg", "png", "mp4", etc. USed to determine the specific
// format of the media type data.
func (m Type) Sub() string {
	groups := mediaTypeRegex.FindStringSubmatch(string(m))
	if len(groups) > 2 {
		return groups[2]
	}
	return ""
}

func (m Type) IsUnknown() bool {
	return m == "" || m == UnknownMimeType
}

func (m Type) IsApplication() bool {
	return m.Top() == Application
}

func (m Type) IsAudio() bool {
	return m.Top() == Audio
}

func (m Type) IsExample() bool {
	return m.Top() == Example
}

func (m Type) IsFont() bool {
	return m.Top() == Font
}

func (m Type) IsHaptics() bool {
	return m.Top() == Haptics
}

func (m Type) IsImage() bool {
	return m.Top() == Image
}

func (m Type) IsMessage() bool {
	return m.Top() == Message
}

func (m Type) IsModel() bool {
	return m.Top() == Model
}

func (m Type) IsMultipart() bool {
	return m.Top() == Multipart
}

func (m Type) IsText() bool {
	return m.Top() == Text
}

func (m Type) IsVideo() bool {
	return m.Top() == Video
}

func Compare(a, b Type) int {
	return strings.Compare(string(a), string(b))
}

// DetectType inspects the first bytes of data and returns the detected media type.
// It is currently a thin wrapper over http.DetectContentType, kept as a seam so that
// callers do not depend directly on net/http and so the detection strategy (e.g.
// swapping in libmagic) can change in the future without touching call sites.
func DetectType(data []byte) Type {
	return Type(http.DetectContentType(data))
}

// iconMap maps a media type to an icon class.
var iconMap = map[Type]string{
	Application:     "far fa-fw fa-file-code",
	Audio:           "far fa-fw fa-file-audio",
	Example:         "far fa-fw fa-file",
	Font:            "fas fa-fw fa-font",
	Haptics:         "far fa-fw fa-file",
	Image:           "far fa-fw fa-file-image",
	Message:         "far fa-fw fa-file",
	Model:           "far fa-fw fa-file",
	Multipart:       "far fa-fw fa-file-lines",
	Text:            "far fa-fw fa-file-alt",
	Video:           "far fa-fw fa-file-video",
	UnknownMimeType: "far fa-fw fa-file",
}

func (m Type) Icon() string {
	if icon, ok := iconMap[Type(m.Top())]; ok {
		return icon
	}
	return iconMap[UnknownMimeType]
}

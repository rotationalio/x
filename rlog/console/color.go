package console

import (
	"fmt"
	"io"
)

const (
	black        uint = 30
	red               = 31
	green             = 32
	yellow            = 33
	blue              = 34
	magenta           = 35
	cyan              = 36
	lightGray         = 37
	darkGray          = 90
	lightRed          = 91
	lightGreen        = 92
	lightYellow       = 93
	lightBlue         = 94
	lightMagenta      = 95
	lightCyan         = 96
	white             = 97
)

func colorize(w io.Writer, color uint8, text string) {
	fmt.Fprintf(w, "\033[%dm%s\033[0m", color, text)
}

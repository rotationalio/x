package console

import (
	"fmt"
	"io"
)

const (
	red         = 31
	cyan        = 36
	lightGray   = 37
	lightRed    = 91
	lightGreen  = 92
	lightYellow = 93
	white       = 97
)

func colorize(w io.Writer, color uint8, text string) {
	fmt.Fprintf(w, "\033[%dm%s\033[0m", color, text)
}

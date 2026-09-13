package urfavehelp

import (
	"os"

	"golang.org/x/term"
)

// TerminalWidth returns the terminal column count, or 0 if w is not an [os.File] terminal.
func TerminalWidth(w any) int {
	f, ok := w.(*os.File)
	if !ok || f == nil {
		return 0
	}
	width, _, err := term.GetSize(int(f.Fd()))
	if err != nil || width <= 0 {
		return 0
	}
	return width
}

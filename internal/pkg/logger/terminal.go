package logger

import "fmt"

type color string

const (
	GREEN  = color("\033[0;32m")
	RED    = color("\033[1;31m")
	YELLOW = color("\033[1;33m")
	CYAN   = color("\033[1;36m")
	NC     = color("\033[0m")
)

func Colorize(c color, msg any) string {
	return fmt.Sprintf("%s%+v%s", c, msg, NC)
}

func Green(msg any) string {
	return Colorize(GREEN, msg)
}

func Red(msg any) string {
	return Colorize(RED, msg)
}

func Yellow(msg any) string {
	return Colorize(YELLOW, msg)
}

func Cyan(msg any) string {
	return Colorize(CYAN, msg)
}

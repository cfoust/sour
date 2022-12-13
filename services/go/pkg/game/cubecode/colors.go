package cubecode

// from engine/rendertext.cpp
const (
	green   = "\f0" // player talk
	blue    = "\f1" // "echo" command
	yellow  = "\f2" // gameplay messages
	red     = "\f3" // important errors
	gray    = "\f4"
	magenta = "\f5"
	orange  = "\f6"
	white   = "\f7"

	save    = "\fs"
	restore = "\fr"
)

func wrap(s, color string) string {
	return save + color + s + restore
}

func Green(s string) string   { return wrap(s, green) }
func Blue(s string) string    { return wrap(s, blue) }
func Yellow(s string) string  { return wrap(s, yellow) }
func Red(s string) string     { return wrap(s, red) }
func Gray(s string) string    { return wrap(s, gray) }
func Magenta(s string) string { return wrap(s, magenta) }
func Orange(s string) string  { return wrap(s, orange) }
func White(s string) string   { return wrap(s, white) }

func Success(s string) string { return Green(s) }
func Fail(s string) string    { return Orange(s) }
func Error(s string) string   { return Red(s) }

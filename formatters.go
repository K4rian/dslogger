package dslogger

import "strings"

// sanitizeLogString replaces control characters (\r, \n, \x00) with escape sequences
// to prevent log injection attacks on the console path.
// Non-ASCII characters and tabs are left untouched.
func sanitizeLogString(s string) string {
	if !strings.ContainsAny(s, "\r\n\x00") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\r':
			b.WriteString(`\r`)
		case '\n':
			b.WriteString(`\n`)
		case 0:
			b.WriteString(`\x00`)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

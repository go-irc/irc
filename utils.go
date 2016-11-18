package irc

import "unicode"

func isNotSpace(r rune) bool {
	return !unicode.IsSpace(r)
}

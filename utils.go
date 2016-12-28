package irc

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

// MaskToRegex converts an irc mask to a go Regexp for more convenient
// use. This should never return an error, but we have this here just
// in case.
func MaskToRegex(rawMask string) (*regexp.Regexp, error) {
	backslashed := false
	unprocessed := rawMask
	buf := &bytes.Buffer{}
	buf.WriteByte('^')

	for len(unprocessed) > 0 {
		i := strings.IndexAny(unprocessed, "\\?*")
		if i < 0 {
			if backslashed {
				buf.WriteString(regexp.QuoteMeta("\\"))
				backslashed = false
			}

			buf.WriteString(regexp.QuoteMeta(unprocessed))
			unprocessed = ""
			break
		}

		if backslashed && i > 0 {
			buf.WriteString(regexp.QuoteMeta("\\"))
			backslashed = false
		} else if backslashed && i == 0 {
			buf.WriteString(regexp.QuoteMeta(string(unprocessed[0])))
			unprocessed = unprocessed[1:]
			backslashed = false
			continue
		}

		buf.WriteString(regexp.QuoteMeta(unprocessed[:i]))

		switch unprocessed[i] {
		case '\\':
			backslashed = true
		case '?':
			buf.WriteString(".")
		case '*':
			buf.WriteString(".*")
		}

		unprocessed = unprocessed[i+1:]
		fmt.Println(unprocessed)
	}

	if backslashed {
		buf.WriteString(regexp.QuoteMeta("\\"))
	}

	buf.WriteByte('$')

	return regexp.Compile(buf.String())
}

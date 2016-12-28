package irc

import (
	"bytes"
	"regexp"
)

// MaskToRegex converts an irc mask to a go Regexp for more convenient
// use. This should never return an error, but we have this here just
// in case.
func MaskToRegex(rawMask string) (*regexp.Regexp, error) {
	input := bytes.NewBufferString(rawMask)

	output := &bytes.Buffer{}
	output.WriteByte('^')

	for {
		c, err := input.ReadByte()
		if err != nil {
			break
		}

		switch c {
		case '\\':
			c, err = input.ReadByte()
			if err != nil {
				output.WriteString(regexp.QuoteMeta("\\"))
				break
			}

			if c == '?' || c == '*' || c == '\\' {
				output.WriteString(regexp.QuoteMeta(string(c)))
			} else {
				output.WriteString(regexp.QuoteMeta("\\" + string(c)))
			}
		case '?':
			output.WriteString(".")
		case '*':
			output.WriteString(".*")
		default:
			output.WriteString(regexp.QuoteMeta(string(c)))
		}
	}

	output.WriteByte('$')

	return regexp.Compile(output.String())
}

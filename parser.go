package irc

import (
	"bytes"
	"strings"
)

// Prefix represents the prefix of a message, generally the user who sent it
type Prefix struct {
	// Name will contain the nick of who sent the message, the
	// server who sent the message, or a blank string
	Name string

	// User will either contain the user who sent the message or a blank string
	User string

	// Host will either contain the host of who sent the message or a blank string
	Host string
}

// Message represents a line parsed from the server
type Message struct {
	// Each message can have a Prefix
	*Prefix

	// Command is which command is being called.
	Command string

	// Params are all the arguments for the command.
	Params []string
}

// ParsePrefix takes an identity string and parses it into an
// identity struct. It will always return an Prefix struct and never
// nil.
func ParsePrefix(line string) *Prefix {
	// Start by creating an Prefix with nothing but the host
	id := &Prefix{
		Name: line,
	}

	uh := strings.SplitN(id.Name, "@", 2)
	if len(uh) == 2 {
		id.Name, id.Host = uh[0], uh[1]
	}

	nu := strings.SplitN(id.Name, "!", 2)
	if len(nu) == 2 {
		id.Name, id.User = nu[0], nu[1]
	}

	return id
}

// Copy will create a new copy of an Prefix
func (p *Prefix) Copy() *Prefix {
	newPrefix := &Prefix{}

	*newPrefix = *p

	return newPrefix
}

// String ensures this is stringable
func (p *Prefix) String() string {
	buf := &bytes.Buffer{}
	buf.WriteString(p.Name)

	if p.User != "" {
		buf.WriteString("!")
		buf.WriteString(p.User)
	}

	if p.Host != "" {
		buf.WriteString("@")
		buf.WriteString(p.Host)
	}

	return buf.String()
}

// ParseMessage takes a message string (usually a whole line) and
// parses it into a Message struct. This will return nil in the case
// of invalid messages.
func ParseMessage(line string) *Message {
	// Trim the line and make sure we have data
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return nil
	}

	c := &Message{Prefix: &Prefix{}}

	if line[0] == ':' {
		split := strings.SplitN(line, " ", 2)
		if len(split) < 2 {
			return nil
		}

		// Parse the identity, if there was one
		c.Prefix = ParsePrefix(string(split[0][1:]))
		line = split[1]
	}

	// Split out the trailing then the rest of the args. Because
	// we expect there to be at least one result as an arg (the
	// command) we don't need to special case the trailing arg and
	// can just attempt a split on " :"
	split := strings.SplitN(line, " :", 2)
	c.Params = strings.FieldsFunc(split[0], func(r rune) bool {
		return r == ' '
	})

	// If there are no args, we need to bail because we need at
	// least the command.
	if len(c.Params) == 0 {
		return nil
	}

	// If we had a trailing arg, append it to the other args
	if len(split) == 2 {
		c.Params = append(c.Params, split[1])
	}

	// Because of how it's parsed, the Command will show up as the
	// first arg.
	c.Command = c.Params[0]
	c.Params = c.Params[1:]

	return c
}

// Trailing returns the last argument in the Message or an empty string
// if there are no args
func (m *Message) Trailing() string {
	if len(m.Params) < 1 {
		return ""
	}

	return m.Params[len(m.Params)-1]
}

// FromChannel is mostly for PRIVMSG messages (and similar derived messages)
// It will check if the message came from a channel or a person.
func (m *Message) FromChannel() bool {
	if len(m.Params) < 1 || len(m.Params[0]) < 1 {
		return false
	}

	switch m.Params[0][0] {
	case '#', '&':
		return true
	default:
		return false
	}
}

// Copy will create a new copy of an message
func (m *Message) Copy() *Message {
	// Create a new message
	newMessage := &Message{}

	// Copy stuff from the old message
	*newMessage = *m

	// Copy the Prefix
	newMessage.Prefix = m.Prefix.Copy()

	// Copy the Params slice
	newMessage.Params = append(make([]string, 0, len(m.Params)), m.Params...)

	return newMessage
}

// String ensures this is stringable
func (m *Message) String() string {
	buf := &bytes.Buffer{}

	// Add the prefix if we have one
	if m.Prefix.Name != "" {
		buf.WriteByte(':')
		buf.WriteString(m.Prefix.String())
		buf.WriteByte(' ')
	}

	// Add the command since we know we'll always have one
	buf.WriteString(m.Command)

	if len(m.Params) > 0 {
		args := m.Params[:len(m.Params)-1]
		trailing := m.Params[len(m.Params)-1]

		if len(args) > 0 {
			buf.WriteByte(' ')
			buf.WriteString(strings.Join(args, " "))
		}

		// If trailing contains a space or starts with a : we
		// need to actually specify that it's trailing.
		if strings.ContainsRune(trailing, ' ') || trailing[0] == ':' {
			buf.WriteString(" :")
		} else {
			buf.WriteString(" ")
		}
		buf.WriteString(trailing)
	}

	return buf.String()
}

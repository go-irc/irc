package irc

import (
	"bytes"
	"fmt"
	"strings"
)

// Identity represents the prefix of a message, generally the user who sent it
type Identity struct {
	// The nick will either contain the nick of who sent the message or a blank string
	Nick string

	// The nick will either contain the user who sent the message or a blank string
	User string

	// The nick will either contain the host of who sent the message or a blank string
	Host string
}

// Event represents a line parsed from the server
type Event struct {
	// Each event can have an identity
	*Identity

	// Command is which command is being called.
	Command string

	// Arguments are all the arguments for the command.
	Args []string
}

// ParseIdentity takes an identity string and parses it into an
// identity struct. It will always return an Identity struct and never
// nil.
func ParseIdentity(line string) *Identity {
	// Start by creating an Identity with nothing but the host
	id := &Identity{
		Host: line,
	}

	uh := strings.SplitN(id.Host, "@", 2)
	if len(uh) != 2 {
		return id
	}
	id.User, id.Host = uh[0], uh[1]

	nu := strings.SplitN(id.User, "!", 2)
	if len(nu) != 2 {
		return id
	}
	id.Nick, id.User = nu[0], nu[1]

	return id
}

// Copy will create a new copy of an Identity
func (i *Identity) Copy() *Identity {
	newIdent := &Identity{}

	*newIdent = *i

	return newIdent
}

// String ensures this is stringable
func (i *Identity) String() string {
	if i.Nick != "" {
		return fmt.Sprintf("%s!%s@%s", i.Nick, i.User, i.Host)
	} else if i.User != "" {
		return fmt.Sprintf("%s@%s", i.User, i.Host)
	}

	return i.Host
}

// ParseEvent takes an event string (usually a whole line) and parses
// it into an Event struct. This will return nil in the case of
// invalid events.
func ParseEvent(line string) *Event {
	// Trim the line and make sure we have data
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return nil
	}

	c := &Event{Identity: &Identity{}}

	if line[0] == ':' {
		split := strings.SplitN(line, " ", 2)
		if len(split) < 2 {
			return nil
		}

		// Parse the identity, if there was one
		c.Identity = ParseIdentity(string(split[0][1:]))
		line = split[1]
	}

	// Split out the trailing then the rest of the args. Because
	// we expect there to be at least one result as an arg (the
	// command) we don't need to special case the trailing arg and
	// can just attempt a split on " :"
	split := strings.SplitN(line, " :", 2)
	c.Args = strings.FieldsFunc(split[0], func(r rune) bool {
		return r == ' '
	})

	// If there are no args, we need to bail because we need at
	// least the command.
	if len(c.Args) == 0 {
		return nil
	}

	// If we had a trailing arg, append it to the other args
	if len(split) == 2 {
		c.Args = append(c.Args, split[1])
	}

	// Because of how it's parsed, the Command will show up as the
	// first arg.
	c.Command = c.Args[0]
	c.Args = c.Args[1:]

	return c
}

// Trailing returns the last argument in the Event or an empty string
// if there are no args
func (e *Event) Trailing() string {
	if len(e.Args) < 1 {
		return ""
	}

	return e.Args[len(e.Args)-1]
}

// FromChannel is mostly for PRIVMSG events (and similar derived events)
// It will check if the event came from a channel or a person.
func (e *Event) FromChannel() bool {
	if len(e.Args) < 1 || len(e.Args[0]) < 1 {
		return false
	}

	switch e.Args[0][0] {
	case '#', '&':
		return true
	default:
		return false
	}
}

// Copy will create a new copy of an event
func (e *Event) Copy() *Event {
	// Create a new event
	newEvent := &Event{}

	// Copy stuff from the old event
	*newEvent = *e

	// Copy the Identity
	newEvent.Identity = e.Identity.Copy()

	// Copy the Args slice
	newEvent.Args = append(make([]string, 0, len(e.Args)), e.Args...)

	return newEvent
}

// String ensures this is stringable
func (e *Event) String() string {
	buf := &bytes.Buffer{}

	// Add the prefix if we have one
	if e.Identity.Host != "" {
		buf.WriteByte(':')
		buf.WriteString(e.Identity.String())
		buf.WriteByte(' ')
	}

	// Add the command since we know we'll always have one
	buf.WriteString(e.Command)

	if len(e.Args) > 0 {
		args := e.Args[:len(e.Args)-1]
		trailing := e.Args[len(e.Args)-1]

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

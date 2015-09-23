package irc

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Client represents a simple IRC client.
type Client struct {
	// Internal things
	currentNick string
	conn        io.ReadWriteCloser
	in          *bufio.Reader
}

// NewClient creates a new Client and sends the initial messages to
// set up nick, user, name and send a password if needed.
func NewClient(rwc io.ReadWriteCloser, nick, user, name, pass string) *Client {
	// Create the client
	c := &Client{
		nick,
		rwc,
		bufio.NewReader(rwc),
	}

	// Send the info we need to
	if len(pass) > 0 {
		c.Writef("PASS %s", pass)
	}

	c.Writef("NICK %s", nick)
	c.Writef("USER %s 0.0.0.0 0.0.0.0 :%s", user, name)

	return c
}

// CurrentNick returns the current nick of the underlying Client.
func (c *Client) CurrentNick() string {
	return c.currentNick
}

// Write is a simple function which will write the given line to the
// underlying connection.
func (c *Client) Write(line string) {
	c.conn.Write([]byte(line))
	c.conn.Write([]byte("\r\n"))
}

// Writef is a wrapper around the client's Write method and
// fmt.Sprintf. Simply use it to send a message as you would normally
// use fmt.Printf.
func (c *Client) Writef(format string, args ...interface{}) {
	c.Write(fmt.Sprintf(format, args...))
}

// WriteMessage writes the given message to the stream
func (c *Client) WriteMessage(m *Message) {
	c.Write(m.String())
}

// ReadMessage returns the next message from the stream or an error.
func (c *Client) ReadMessage() (*Message, error) {
	line, err := c.in.ReadString('\n')
	if err != nil {
		return nil, err
	}

	// Parse the message from our line
	m := ParseMessage(line)

	// Now that we have the message parsed, do some preprocessing on it
	lastArg := m.Trailing()

	// Clean up CTCP stuff so everyone
	// doesn't have to parse it manually
	if m.Command == "PRIVMSG" && len(lastArg) > 0 && lastArg[0] == '\x01' {
		m.Command = "CTCP"

		if i := strings.LastIndex(lastArg, "\x01"); i > -1 {
			m.Params[len(m.Params)-1] = lastArg[1:i]
		}
	} else if m.Command == "PING" {
		c.Writef("PONG :%s", lastArg)
	} else if m.Command == "NICK" {
		if m.Prefix.Name == c.currentNick && len(m.Params) > 0 {
			c.currentNick = m.Params[0]
		}
	} else if m.Command == "001" {
		c.currentNick = m.Params[0]
	} else if m.Command == "437" || m.Command == "433" {
		c.currentNick = c.currentNick + "_"
		c.Writef("NICK %s", c.currentNick)
	}

	return m, nil
}

// Reply to a Message with a convenience wrapper around Writef
func (c *Client) Reply(m *Message, format string, v ...interface{}) error {
	// Sanity check
	if len(m.Params) < 1 || len(m.Params[0]) < 1 {
		return errors.New("Invalid IRC message")
	}

	if m.FromChannel() {
		v = prepend(m.Params[0], v)
		c.Writef("PRIVMSG %s :"+format, v...)
	} else {
		v = prepend(m.Prefix.Name, v)
		c.Writef("PRIVMSG %s :"+format, v...)
	}

	return nil
}

// MentionReply acts the same as Reply but it will prefix the message
// with the user's name if the message came from a channel.
func (c *Client) MentionReply(m *Message, format string, v ...interface{}) error {
	// Sanity check
	if len(m.Params) < 1 || len(m.Params[0]) < 1 {
		return errors.New("Invalid IRC message")
	}

	if m.FromChannel() {
		format = "%s: " + format
		v = prepend(m.Prefix.Name, v)
	}

	return c.Reply(m, format, v...)
}

// CTCPReply is a convenience function to respond to CTCP requests.
func (c *Client) CTCPReply(m *Message, format string, v ...interface{}) error {
	// Sanity check
	if len(m.Params) < 1 || len(m.Params[0]) < 1 {
		return errors.New("Invalid IRC message")
	}

	v = prepend(m.Prefix.Name, v)
	c.Writef("NOTICE %s :\x01"+format+"\x01", v...)
	return nil
}

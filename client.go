package irc

import (
	"errors"
	"io"
)

// Client represents a simple IRC client.
type Client struct {
	*Conn

	// NickCollisionCallback is a simple callback which returns a
	// replacement nick for the given client on nick collision.
	NickCollisionCallback func(c *Client) string

	// Internal things
	currentNick string
}

// NewClient creates a new Client and sends the initial messages to
// set up nick, user, name and send a password if needed.
func NewClient(rwc io.ReadWriteCloser, nick, user, name, pass string) *Client {
	// Create the client
	c := &Client{
		NewConn(rwc),
		func(c *Client) string {
			return c.CurrentNick() + "_"
		},
		nick,
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

// ReadMessage returns the next message from the stream or an error.
func (c *Client) ReadMessage() (*Message, error) {
	m, err := c.Conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	if m.Command == "NICK" {
		if m.Prefix.Name == c.currentNick && len(m.Params) > 0 {
			c.currentNick = m.Params[0]
		}
	} else if m.Command == "PING" {
		c.Writef("PONG :%s", m.Trailing())
	} else if m.Command == "001" {
		c.currentNick = m.Params[0]
	} else if m.Command == "437" || m.Command == "433" {
		c.currentNick = c.NickCollisionCallback(c)
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

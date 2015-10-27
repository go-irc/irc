package irc

import "io"

// Client is a wrapper around a Conn which handles a number of the
// annoyances and things which will probably be common among many clients.
type Client struct {
	*Conn
	Handler     Handler
	currentNick string
}

// NewClient creates a new Client given an io.ReadWriteCloser and a
// Handler.
func NewClient(rwc io.ReadWriteCloser, handler Handler) *Client {
	c := &Client{
		NewConn(rwc),
		handler,
		"",
	}

	return c
}

// CurrentNick returns the current nick associated with this client.
func (c *Client) CurrentNick() string {
	return c.currentNick
}

// Run starts the main loop for this client.
func (c *Client) Run(nick, user, name, pass string) error {
	c.currentNick = nick

	var err error
	var m *Message
	if pass != "" {
		c.Writef("PASS :%s", pass)
	}

	c.Writef("NICK :%s", nick)
	c.Writef("USER %s 0.0.0.0 0.0.0.0 :%s", user, name)

	for {
		m, err = c.ReadMessage()
		if err != nil {
			break
		}

		if m.Command == "NICK" {
			if m.Prefix.Name == c.currentNick && len(m.Params) > 0 {
				c.currentNick = m.Params[0]
			}
		} else if m.Command == "PING" {
			c.Writef("PONG :%s", m.Trailing())
		} else if m.Command == "001" {
			c.currentNick = m.Params[0]
		} else if m.Command == "433" || m.Command == "437" {
			c.currentNick = c.currentNick + "_"
			c.Writef("NICK :%s", c.currentNick)
		}

		c.Handler.Handle(c, m)
	}

	return err
}

package irc

import (
	"fmt"
	"io"
)

// ClientConfig is a structure used to configure a Client.
type ClientConfig struct {
	// General connection information.
	Nick string
	Pass string
	User string
	Name string

	// Handler is used for message dispatching.
	Handler Handler

	// OutputFilter is used to limit which messages get sent.
	OutputFilter Filter
}

// Client is a wrapper around Conn which is designed to make common operations
// much simpler.
type Client struct {
	*Conn
	config ClientConfig

	// Internal state
	currentNick string
}

// NewClient creates a client given an io stream and a client config.
func NewClient(rwc io.ReadWriteCloser, config ClientConfig) *Client {
	return &Client{
		Conn:   NewConn(rwc),
		config: config,
	}
}

// Run starts the main loop for this IRC connection. Note that it may break in
// strange and unexpected ways if it is called again before the first connection
// exits.
func (c *Client) Run() error {
	c.currentNick = c.config.Nick

	if c.config.Pass != "" {
		c.Conn.Writef("PASS :%s", c.config.Pass)
	}

	c.Conn.Writef("NICK :%s", c.config.Nick)
	c.Conn.Writef("USER %s 0.0.0.0 0.0.0.0 :%s", c.config.User, c.config.Name)

	for {
		m, err := c.ReadMessage()
		if err != nil {
			return err
		}

		switch m.Command {
		case "PING":
			reply := m.Copy()
			reply.Command = "PONG"
			c.Conn.WriteMessage(reply)
		case "NICK":
			if m.Prefix.Name == c.currentNick && len(m.Params) > 0 {
				c.currentNick = m.Params[0]
			}
		case "001":
			c.currentNick = m.Params[0]
		case "433", "437":
			c.currentNick = c.currentNick + "_"
			c.Conn.Writef("NICK :%s", c.currentNick)
		}

		if c.config.Handler != nil {
			c.config.Handler.Handle(c, m)
		}
	}
}

// shouldFilterMessage assumes c.config.OutputFilter is not nil. It
// will return true if the message should not be sent.
func (c *Client) shouldFilterMessage(m *Message) bool {
	if c.config.OutputFilter.Filter(c, m) {
		if c.Conn.DebugCallback != nil {
			c.Conn.DebugCallback("filtered", m.String())
		}

		return true
	}

	return false
}

// WriteMessage is a wrapper around (*Conn).Write which calls the
// output filter.
func (c *Client) Write(line string) {
	if c.config.OutputFilter != nil {
		m := ParseMessage(line)
		if c.shouldFilterMessage(m) {
			return
		}
	}

	c.Conn.Write(line)
}

// Writef is a wrapper around (*Conn).Writef which calls the output
// filter.
func (c *Client) Writef(format string, args ...interface{}) {
	if c.config.OutputFilter != nil {
		m := ParseMessage(fmt.Sprintf(format, args...))
		if c.shouldFilterMessage(m) {
			return
		}
	}

	c.Conn.Writef(format, args...)
}

// WriteMessage is a wrapper around (*Conn).WriteMessage which calls
// the output filter.
func (c *Client) WriteMessage(m *Message) {
	if c.config.OutputFilter != nil {
		if c.shouldFilterMessage(m) {
			return
		}
	}

	c.Conn.WriteMessage(m)
}

// CurrentNick returns what the nick of the client is known to be at this point
// in time.
func (c *Client) CurrentNick() string {
	return c.currentNick
}

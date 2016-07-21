package irc

import (
	"crypto/tls"
	"io"
	"net"
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

// Dial will connect to the host and create a client with the given
// ClientConfig.
func Dial(host string, config ClientConfig) (*Client, error) {
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return nil, err
	}

	return NewClient(conn, config), nil
}

// DialTLS will connect to the host using the tlsConf and create a client with
// the given ClientConfig.
func DialTLS(host string, tlsConf *tls.Config, config ClientConfig) (*Client, error) {
	conn, err := tls.Dial("tcp", host, tlsConf)
	if err != nil {
		return nil, err
	}

	return NewClient(conn, config), nil
}

// Run starts the main loop for this IRC connection. Note that it may break in
// strange and unexpected ways if it is called again before the first connection
// exits.
func (c *Client) Run() error {
	c.currentNick = c.config.Nick

	if c.config.Pass != "" {
		c.Writef("PASS :%s", c.config.Pass)
	}

	c.Writef("NICK :%s", c.config.Nick)
	c.Writef("USER %s 0.0.0.0 0.0.0.0 :%s", c.config.User, c.config.Name)

	for {
		m, err := c.ReadMessage()
		if err != nil {
			return err
		}

		switch m.Command {
		case "PING":
			reply := m.Copy()
			reply.Command = "PONG"
			c.WriteMessage(reply)
		case "NICK":
			if m.Prefix.Name == c.currentNick && len(m.Params) > 0 {
				c.currentNick = m.Params[0]
			}
		case "001":
			c.currentNick = m.Params[0]
		case "433", "437":
			c.currentNick = c.currentNick + "_"
			c.Writef("NICK :%s", c.currentNick)
		}

		if c.config.Handler != nil {
			c.config.Handler.Handle(c, m)
		}
	}
}

// CurrentNick returns what the nick of the client is known to be at this point
// in time.
func (c *Client) CurrentNick() string {
	return c.currentNick
}

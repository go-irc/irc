package irc

import (
	"io"
	"sync"
	"time"
)

// clientFilters are pre-processing which happens for certain message
// types. These were moved from below to keep the complexity of each
// component down.
var clientFilters = map[string]func(*Client, *Message){
	"001": func(c *Client, m *Message) {
		c.currentNick = m.Params[0]
	},
	"433": func(c *Client, m *Message) {
		c.currentNick = c.currentNick + "_"
		c.Writef("NICK :%s", c.currentNick)
	},
	"437": func(c *Client, m *Message) {
		c.currentNick = c.currentNick + "_"
		c.Writef("NICK :%s", c.currentNick)
	},
	"PING": func(c *Client, m *Message) {
		reply := m.Copy()
		reply.Command = "PONG"
		c.WriteMessage(reply)
	},
	"PRIVMSG": func(c *Client, m *Message) {
		// Clean up CTCP stuff so everyone doesn't have to parse it
		// manually.
		lastArg := m.Trailing()
		lastIdx := len(lastArg) - 1
		if lastIdx > 0 && lastArg[0] == '\x01' && lastArg[lastIdx] == '\x01' {
			m.Command = "CTCP"
			m.Params[len(m.Params)-1] = lastArg[1:lastIdx]
		}
	},
	"NICK": func(c *Client, m *Message) {
		if m.Prefix.Name == c.currentNick && len(m.Params) > 0 {
			c.currentNick = m.Params[0]
		}
	},
}

// ClientConfig is a structure used to configure a Client.
type ClientConfig struct {
	// General connection information.
	Nick string
	Pass string
	User string
	Name string

	// Connection settings
	PingFrequency time.Duration
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration

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
func NewClient(rwc io.ReadWriter, config ClientConfig) *Client {
	c := &Client{
		Conn:   NewConn(rwc),
		config: config,
	}
	c.Reader.SetTimeout(config.ReadTimeout)
	c.Writer.SetTimeout(config.WriteTimeout)
	return c
}

// Run starts the main loop for this IRC connection. Note that it may break in
// strange and unexpected ways if it is called again before the first connection
// exits.
func (c *Client) Run() error {
	// exiting is used by the main goroutine here to ensure any sub-goroutines
	// get closed when exiting.
	exiting := make(chan struct{})
	var wg sync.WaitGroup

	// If PingFrequency isn't the zero value, we need to start a ping goroutine.
	if c.config.PingFrequency > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			t := time.NewTicker(c.config.PingFrequency)
			defer t.Stop()

			for {
				select {
				case <-t.C:
					c.Writef("PING :%d", time.Now().Unix())
				case <-exiting:
					break
				}
			}
		}()
	}

	c.currentNick = c.config.Nick

	if c.config.Pass != "" {
		c.Writef("PASS :%s", c.config.Pass)
	}

	c.Writef("NICK :%s", c.config.Nick)
	c.Writef("USER %s 0.0.0.0 0.0.0.0 :%s", c.config.User, c.config.Name)

	var err error
	var m *Message
	for {
		m, err = c.ReadMessage()
		if err != nil {
			break
		}

		if f, ok := clientFilters[m.Command]; ok {
			f(c, m)
		}

		if c.config.Handler != nil {
			c.config.Handler.Handle(c, m)
		}
	}

	close(exiting)
	wg.Wait()

	return err
}

// CurrentNick returns what the nick of the client is known to be at this point
// in time.
func (c *Client) CurrentNick() string {
	return c.currentNick
}

// FromChannel takes a Message representing a PRIVMSG and returns if that
// message came from a channel or directly from a user.
func (c *Client) FromChannel(m *Message) bool {
	if len(m.Params) < 1 {
		return false
	}

	// The first param is the target, so if this doesn't match the current nick,
	// the message came from a channel.
	return m.Params[0] != c.currentNick
}

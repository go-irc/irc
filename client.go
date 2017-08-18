package irc

import (
	"errors"
	"fmt"
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
	"PONG": func(c *Client, m *Message) {
		if c.config.PingFrequency > 0 {
			c.sentPingLock.Lock()
			defer c.sentPingLock.Unlock()

			// If there haven't been any sent pings, so we can safely ignore
			// this pong.
			if len(c.sentPings) == 0 {
				return
			}

			if fmt.Sprintf("%d", c.sentPings[0].Unix()) == m.Trailing() {
				c.sentPings = c.sentPings[1:]
			}
		}
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
	PingTimeout   time.Duration

	// SendLimit is how frequent messages can be sent. If this is zero,
	// there will be no limit.
	SendLimit time.Duration

	// SendBurst is the number of messages which can be sent in a burst.
	SendBurst int

	// Handler is used for message dispatching.
	Handler Handler
}

// Client is a wrapper around Conn which is designed to make common operations
// much simpler.
type Client struct {
	*Conn
	rwc    io.ReadWriteCloser
	config ClientConfig

	// Internal state
	currentNick  string
	sentPingLock sync.Mutex
	sentPings    []time.Time

	limitTick *time.Ticker
	limiter   chan struct{}
	tickDone  chan struct{}
}

// NewClient creates a client given an io stream and a client config.
func NewClient(rwc io.ReadWriteCloser, config ClientConfig) *Client {
	c := &Client{
		Conn:     NewConn(rwc),
		rwc:      rwc,
		config:   config,
		tickDone: make(chan struct{}),
	}

	// Replace the writer writeCallback with one of our own
	c.Conn.Writer.writeCallback = c.writeCallback

	return c
}

func (c *Client) writeCallback(w *Writer, line string) error {
	if c.limiter != nil {
		<-c.limiter
	}

	_, err := w.writer.Write([]byte(line + "\r\n"))
	return err
}

func (c *Client) maybeStartLimiter() {
	if c.config.SendLimit == 0 {
		return
	}

	// If SendBurst is 0, this will be unbuffered, so keep that in mind.
	c.limiter = make(chan struct{}, c.config.SendBurst)

	c.limitTick = time.NewTicker(c.config.SendLimit)

	go func() {
		var done bool
		for !done {
			select {
			case <-c.limitTick.C:
				select {
				case c.limiter <- struct{}{}:
				default:
				}
			case <-c.tickDone:
				done = true
			}
		}

		c.limitTick.Stop()
		close(c.limiter)
		c.limiter = nil
		c.tickDone <- struct{}{}
	}()
}

func (c *Client) stopLimiter() {
	if c.limiter == nil {
		return
	}

	c.tickDone <- struct{}{}
	<-c.tickDone
}

func (c *Client) startPingLoop(wg *sync.WaitGroup, errChan chan error, exiting chan struct{}) {
	// We're firing off two new goroutines here.
	wg.Add(2)

	// PING ticker
	go func() {
		defer wg.Done()

		t := time.NewTicker(c.config.PingFrequency)
		defer t.Stop()

		for {
			select {
			case <-t.C:
				timestamp := time.Now()

				// We need to append before we write so we can guarantee
				// this will be in the queue when the PONG gets here.
				c.sentPingLock.Lock()
				c.sentPings = append(c.sentPings, timestamp)
				c.sentPingLock.Unlock()

				err := c.Writef("PING :%d", timestamp.Unix())
				if err != nil {
					errChan <- err
					c.rwc.Close()
					return
				}
			case <-exiting:
				return
			}
		}
	}()

	// PONG checker
	go func() {
		defer wg.Done()

		var timer *time.Timer
		var pingSent bool

		for {
			c.sentPingLock.Lock()
			pingSent = len(c.sentPings) > 0
			if pingSent {
				timer = time.NewTimer(c.config.PingTimeout)
			} else {
				timer = time.NewTimer(c.config.PingFrequency)
			}
			c.sentPingLock.Unlock()

			select {
			case <-timer.C:
				if pingSent {
					errChan <- errors.New("PING timeout")
					c.rwc.Close()
					return
				}
			case <-exiting:
				return
			}

			timer.Stop()
		}
	}()
}

// Run starts the main loop for this IRC connection. Note that it may break in
// strange and unexpected ways if it is called again before the first connection
// exits.
func (c *Client) Run() error {
	c.maybeStartLimiter()
	defer c.stopLimiter()

	// exiting is used by the main goroutine here to ensure any sub-goroutines
	// get closed when exiting.
	exiting := make(chan struct{})
	errChan := make(chan error, 3)
	var wg sync.WaitGroup

	// If PingFrequency isn't the zero value, we need to start a ping goroutine
	// and a pong checker goroutine.
	if c.config.PingFrequency > 0 {
		c.startPingLoop(&wg, errChan, exiting)
	}

	c.currentNick = c.config.Nick

	if c.config.Pass != "" {
		c.Writef("PASS :%s", c.config.Pass)
	}

	c.Writef("NICK :%s", c.config.Nick)
	c.Writef("USER %s 0.0.0.0 0.0.0.0 :%s", c.config.User, c.config.Name)

	for {
		m, err := c.ReadMessage()
		if err != nil {
			errChan <- err
			break
		}

		if f, ok := clientFilters[m.Command]; ok {
			f(c, m)
		}

		if c.config.Handler != nil {
			c.config.Handler.Handle(c, m)
		}
	}

	// Wait for an error from any goroutine, then signal we're exiting and wait
	// for the goroutines to exit.
	err := <-errChan
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

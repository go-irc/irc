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
		if c.incomingPongChan != nil {
			select {
			case c.incomingPongChan <- m.Trailing():
			default:
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
	config ClientConfig

	// Internal state
	currentNick      string
	limiter          chan struct{}
	incomingPongChan chan string
}

// NewClient creates a client given an io stream and a client config.
func NewClient(rw io.ReadWriter, config ClientConfig) *Client {
	c := &Client{
		Conn:   NewConn(rw),
		config: config,
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

func (c *Client) maybeStartLimiter(wg *sync.WaitGroup, errChan chan error, exiting chan struct{}) {
	if c.config.SendLimit == 0 {
		return
	}

	wg.Add(1)

	// If SendBurst is 0, this will be unbuffered, so keep that in mind.
	c.limiter = make(chan struct{}, c.config.SendBurst)
	limitTick := time.NewTicker(c.config.SendLimit)

	go func() {
		defer wg.Done()

		var done bool
		for !done {
			select {
			case <-limitTick.C:
				select {
				case c.limiter <- struct{}{}:
				default:
				}
			case <-exiting:
				done = true
			}
		}

		limitTick.Stop()
		close(c.limiter)
		c.limiter = nil
	}()
}

func (c *Client) maybeStartPingLoop(wg *sync.WaitGroup, errChan chan error, exiting chan struct{}) {
	if c.config.PingFrequency <= 0 {
		return
	}

	wg.Add(1)
	c.incomingPongChan = make(chan string, 5)
	go c.pingLoop(wg, errChan, exiting)
}

type pingDeadline struct {
	Data     string
	Deadline <-chan time.Time
}

func (c *Client) pingLoop(wg *sync.WaitGroup, errChan chan error, exiting chan struct{}) {
	defer wg.Done()

	var (
		sentPings       = map[string]time.Time{}
		pingDeadlines   []pingDeadline
		currentDeadline pingDeadline
		ticker          = time.NewTicker(c.config.PingFrequency)
	)

	defer ticker.Stop()

	for {
		if len(pingDeadlines) > 0 {
			currentDeadline = pingDeadlines[0]
			pingDeadlines = pingDeadlines[1:]
		}

		select {
		case <-ticker.C:
			timestamp := time.Now()
			err := c.Writef("PING :%d", timestamp.Unix())
			if err != nil {
				errChan <- err
				return
			}
			deadline := pingDeadline{
				Data:     fmt.Sprintf("%d", timestamp.Unix()),
				Deadline: time.After(c.config.PingTimeout),
			}
			sentPings[deadline.Data] = timestamp
			pingDeadlines = append(pingDeadlines, deadline)
		case <-currentDeadline.Deadline:
			if _, ok := sentPings[currentDeadline.Data]; ok {
				errChan <- errors.New("PING timeout")
			}
			currentDeadline.Deadline = nil
			return
		case data := <-c.incomingPongChan:
			delete(sentPings, data)
		case <-exiting:
			return
		}
	}
}

// Run starts the main loop for this IRC connection. Note that it may break in
// strange and unexpected ways if it is called again before the first connection
// exits.
func (c *Client) Run() error {
	// exiting is used by the main goroutine here to ensure any sub-goroutines
	// get closed when exiting.
	exiting := make(chan struct{})
	errChan := make(chan error, 3)
	var wg sync.WaitGroup

	c.maybeStartLimiter(&wg, errChan, exiting)
	c.maybeStartPingLoop(&wg, errChan, exiting)

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

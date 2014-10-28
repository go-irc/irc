package irc

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	tomb "gopkg.in/tomb.v2"
)

type Client struct {
	Logger *log.Logger

	handler     Handler
	connected   bool
	currentNick string
	nick        string
	user        string
	name        string
	password    string
	lock        *sync.Mutex
	conn        io.ReadWriteCloser
	t           *tomb.Tomb
	write       chan string
}

func NewClient(handler Handler, nick string, user string, name string, pass string) *Client {
	// Create the client
	c := &Client{
		nil,
		handler,
		false,
		nick,
		nick,
		user,
		name,
		pass,
		&sync.Mutex{},
		nil,
		&tomb.Tomb{},
		make(chan string),
	}

	return c
}

func (c *Client) CurrentNick() string {
	return c.currentNick
}

func (c *Client) Dial(host string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.connected {
		return errors.New("Already connected")
	}

	var err error
	c.conn, err = net.Dial("tcp", host)
	if err != nil {
		return err
	}

	c.connected = true

	return c.start()
}

func (c *Client) DialTLS(host string, conf *tls.Config) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.connected {
		return errors.New("Already connected")
	}

	var err error
	c.conn, err = tls.Dial("tcp", host, conf)
	if err != nil {
		return err
	}

	c.connected = true

	return c.start()
}

func (c *Client) Write(line string) {
	// Try to write it to the writer. Fall back to waiting until the bot dies.
	select {
	case c.write <- line + "\r\n":
	case <-c.t.Dying():
	}
}

func (c *Client) Writef(format string, args ...interface{}) {
	c.Write(fmt.Sprintf(format, args...))
}

func (c *Client) start() error {
	// Start up the tomb with all our loops.
	//
	// Note that it's only safe to call tomb.Go from inside
	// other functions that have been started the same way,
	// so we make a quick closure to take care of that.
	c.t.Go(func() error {
		// Ping Loop
		c.t.Go(c.pingLoop)

		// Read Loop
		c.t.Go(c.readLoop)

		// Write Loop
		c.t.Go(c.writeLoop)

		// Cleanup Loop
		c.t.Go(c.cleanupLoop)

		// Actually connect
		if len(c.password) > 0 {
			c.Writef("PASS %s", c.password)
		}

		c.Writef("NICK %s", c.nick)
		c.Writef("USER %s 0.0.0.0 0.0.0.0 :%s", c.user, c.name)

		return nil
	})

	// This will wait until all goroutines in the Tomb die
	return c.t.Wait()
}

func (c *Client) pingLoop() error {
	// Tick every 2 minutes
	t := time.NewTicker(2 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			c.Writef("PING :%d", time.Now().UnixNano())
		case <-c.t.Dying():
			return nil
		}
	}
}

func (c *Client) readLoop() error {
	in := bufio.NewReader(c.conn)
	for {
		// If we're dying exit out
		select {
		case <-c.t.Dying():
			return nil
		default:
		}

		line, err := in.ReadString('\n')
		if err != nil {
			return err
		}

		if c.Logger != nil {
			c.Logger.Printf("<-- %s", line)
		}

		// Parse the event from our line
		e := ParseEvent(line)

		// Now that we have the event parsed, do some preprocessing on it
		lastArg := e.Trailing()

		// Clean up CTCP stuff so everyone
		// doesn't have to parse it manually
		if e.Command == "PRIVMSG" && len(lastArg) > 0 && lastArg[0] == '\x01' {
			e.Command = "CTCP"

			if i := strings.LastIndex(lastArg, "\x01"); i > -1 {
				e.Args[len(e.Args)-1] = lastArg[1:i]
			}
		} else if e.Command == "PING" {
			c.Writef("PONG :%s", lastArg)
		} else if e.Command == "PONG" {
			ns, _ := strconv.ParseInt(lastArg, 10, 64)
			delta := time.Duration(time.Now().UnixNano() - ns)

			if c.Logger != nil {
				fmt.Printf("!!! Lag: %v\n", delta)
			}
		} else if e.Command == "NICK" {
			if e.Identity.Nick == c.currentNick {
				c.currentNick = lastArg
			}
		} else if e.Command == "001" {
			c.currentNick = e.Args[0]
		} else if e.Command == "437" || e.Command == "433" {
			c.currentNick = c.currentNick + "_"
			c.Writef("NICK %s", c.currentNick)
		}

		c.handler.HandleEvent(c, e)
	}
}

func (c *Client) writeLoop() error {
	// Set up a rate limiter
	// Based on https://code.google.com/p/go-wiki/wiki/RateLimiting
	// 2 lps with a burst of 5
	throttle := make(chan time.Time, 5)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	go func() {
		for ns := range ticker.C {
			select {
			case throttle <- ns:
			default:
			}
		}
	}()

	for {
		select {
		case line := <-c.write:
			select {
			case <-throttle:
				if c.Logger != nil {
					fmt.Printf("--> %s", line)
				}
				c.conn.Write([]byte(line))
			case <-c.t.Dying():
			}
		case <-c.t.Dying():
			return nil
		}
	}
}

func (c *Client) cleanupLoop() error {
	select {
	case <-c.t.Dying():
		c.conn.Close()
	}
	return nil
}

func prepend(e interface{}, v []interface{}) []interface{} {
	var vc []interface{}

	vc = append(vc, e)
	vc = append(vc, v...)

	return vc
}

func (c *Client) MentionReply(e *Event, format string, v ...interface{}) error {
	// Sanity check
	if len(e.Args) < 1 || len(e.Args[0]) < 1 {
		return errors.New("Invalid IRC event")
	}

	if e.FromChannel() {
		format = "%s: " + format

		v = prepend(e.Identity.Nick, v)
	}

	return c.Reply(e, format, v...)
}

func (c *Client) CTCPReply(e *Event, format string, v ...interface{}) error {
	if len(e.Args) < 1 || len(e.Args[0]) < 1 {
		return errors.New("Invalid IRC event")
	}

	v = prepend(e.Identity.Nick, v)
	c.Writef("NOTICE %s :\x01"+format+"\x01", v...)
	return nil
}

func (c *Client) Reply(e *Event, format string, v ...interface{}) error {
	// Sanity check
	if len(e.Args) < 1 || len(e.Args[0]) < 1 {
		return errors.New("Invalid IRC event")
	}

	if e.FromChannel() {
		v = prepend(e.Args[0], v)
		c.Writef("PRIVMSG %s :"+format, v...)
	} else {
		v = prepend(e.Identity.Nick, v)
		c.Writef("PRIVMSG %s :"+format, v...)
	}

	return nil
}

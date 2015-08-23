package irc

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	// Logger for messages. By defaut this will be a NilLogger
	Logger Logger

	// Internal things
	currentNick string
	conn        io.ReadWriteCloser
	in          *bufio.Reader
}

func Dial(addr string, nick, user, name, pass string) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	return NewClient(conn, nick, user, name, pass), nil
}

func DialTLS(addr string, c *tls.Config, nick, user, name, pass string) (*Client, error) {
	conn, err := tls.Dial("tcp", addr, c)
	if err != nil {
		return nil, err
	}

	return NewClient(conn, nick, user, name, pass), nil
}

func NewClient(rwc io.ReadWriteCloser, nick, user, name, pass string) *Client {
	// Create the client
	c := &Client{
		&NilLogger{},
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

func (c *Client) CurrentNick() string {
	return c.currentNick
}

func (c *Client) Write(line string) {
	if c.Logger != nil {
		c.Logger.Debug("-->", line)
	}
	c.conn.Write([]byte(line))
	c.conn.Write([]byte("\r\n"))
}

func (c *Client) Writef(format string, args ...interface{}) {
	c.Write(fmt.Sprintf(format, args...))
}

func (c *Client) ReadEvent() (*Event, error) {
	line, err := c.in.ReadString('\n')
	if err != nil {
		return nil, err
	}

	if c.Logger != nil {
		c.Logger.Debug("<--", strings.TrimRight(line, "\r\n"))
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
			c.Logger.Info("!!! Lag:", delta)
		}
	} else if e.Command == "NICK" {
		if e.Identity.Nick == c.currentNick && len(e.Args) > 0 {
			c.currentNick = e.Args[0]
		}
	} else if e.Command == "001" {
		c.currentNick = e.Args[0]
	} else if e.Command == "437" || e.Command == "433" {
		c.currentNick = c.currentNick + "_"
		c.Writef("NICK %s", c.currentNick)
	}

	return e, nil
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
	// Sanity check
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

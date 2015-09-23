package irc

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Conn represents a simple IRC client.
type Conn struct {
	// Internal things
	conn io.ReadWriteCloser
	in   *bufio.Reader
}

// NewConn creates a new Conn
func NewConn(rwc io.ReadWriteCloser) *Conn {
	// Create the client
	c := &Conn{
		rwc,
		bufio.NewReader(rwc),
	}

	return c
}

// Write is a simple function which will write the given line to the
// underlying connection.
func (c *Conn) Write(line string) {
	c.conn.Write([]byte(line))
	c.conn.Write([]byte("\r\n"))
}

// Writef is a wrapper around the connection's Write method and
// fmt.Sprintf. Simply use it to send a message as you would normally
// use fmt.Printf.
func (c *Conn) Writef(format string, args ...interface{}) {
	c.Write(fmt.Sprintf(format, args...))
}

// WriteMessage writes the given message to the stream
func (c *Conn) WriteMessage(m *Message) {
	c.Write(m.String())
}

// ReadMessage returns the next message from the stream or an error.
func (c *Conn) ReadMessage() (*Message, error) {
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
	}

	return m, nil
}

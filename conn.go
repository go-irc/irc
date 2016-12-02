package irc

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Conn represents a simple IRC client.
type Conn struct {
	// DebugCallback is a callback for every line of input and
	// output. It is meant for debugging and is not guaranteed to
	// be stable.
	DebugCallback func(operation, line string)

	// Internal things
	conn io.ReadWriteCloser
	in   *bufio.Reader
}

// NewConn creates a new Conn
func NewConn(rwc io.ReadWriteCloser) *Conn {
	// Create the client
	c := &Conn{
		func(operation, line string) {},
		rwc,
		bufio.NewReader(rwc),
	}

	return c
}

// Write is a simple function which will write the given line to the
// underlying connection.
func (c *Conn) Write(line string) error {
	c.DebugCallback("write", line)
	_, err := c.conn.Write([]byte(line + "\r\n"))
	return err
}

// Writef is a wrapper around the connection's Write method and
// fmt.Sprintf. Simply use it to send a message as you would normally
// use fmt.Printf.
func (c *Conn) Writef(format string, args ...interface{}) error {
	return c.Write(fmt.Sprintf(format, args...))
}

// WriteMessage writes the given message to the stream
func (c *Conn) WriteMessage(m *Message) error {
	return c.Write(m.String())
}

// ReadMessage returns the next message from the stream or an error.
func (c *Conn) ReadMessage() (*Message, error) {
	line, err := c.in.ReadString('\n')
	if err != nil {
		return nil, err
	}

	c.DebugCallback("read", strings.TrimRight(line, "\r\n"))

	// Parse the message from our line
	return ParseMessage(line)
}

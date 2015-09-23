package irc

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Conn represents a simple IRC client.
type Conn struct {
	// DebugCallback is a callback for every line of input and
	// output. It is meant for debugging and is not guaranteed to
	// be stable.
	DebugCallback func(line string)

	// InputHandler is used with Conn.Run to dispatch incoming
	// messages.
	InputHandler InputHandler

	// OutputHandler is used with Conn.Write to filter outgoing
	// messages.
	OutputHandler OutputHandler

	// Internal things
	conn io.ReadWriteCloser
	in   *bufio.Reader
}

// NewConn creates a new Conn
func NewConn(rwc io.ReadWriteCloser) *Conn {
	// Create the client
	c := &Conn{
		func(line string) {},
		nil,
		nil,
		rwc,
		bufio.NewReader(rwc),
	}

	return c
}

// Write is a simple function which will write the given line to the
// underlying connection.
//
// This is very low level and should probably not be used very often
// as it will bypass the OutputHandler.
func (c *Conn) Write(line string) {
	c.DebugCallback("--> " + line)
	c.conn.Write([]byte(line))
	c.conn.Write([]byte("\r\n"))
}

// Writef is a wrapper around the connection's Write method and
// fmt.Sprintf. Simply use it to send a message as you would normally
// use fmt.Printf.
//
// This is very low level and should probably not be used very often
// as it will bypass the OutputHandler.
func (c *Conn) Writef(format string, args ...interface{}) {
	c.Write(fmt.Sprintf(format, args...))
}

// WriteMessage writes the given message to the stream. This will run
// all output through the OutputHandler.
func (c *Conn) WriteMessage(m *Message) {
	messages := []*Message{m}
	if c.OutputHandler != nil {
		messages = c.OutputHandler.HandleOutput(c, m)
	}

	for _, msg := range messages {
		c.Write(msg.String())
	}
}

// ReadMessage returns the next message from the stream or an error.
func (c *Conn) ReadMessage() (*Message, error) {
	line, err := c.in.ReadString('\n')
	if err != nil {
		return nil, err
	}

	c.DebugCallback("<-- " + strings.TrimRight(line, "\r\n"))

	// Parse the message from our line
	m := ParseMessage(line)

	// Now that we have the message parsed, do some preprocessing
	// on it
	lastArg := m.Trailing()

	// Clean up CTCP stuff so everyone doesn't have to parse it
	// manually
	if m.Command == "PRIVMSG" && len(lastArg) > 0 && lastArg[0] == '\x01' {
		m.Command = "CTCP"

		if i := strings.LastIndex(lastArg, "\x01"); i > -1 {
			m.Params[len(m.Params)-1] = lastArg[1:i]
		}
	}

	return m, nil
}

// Run is a simple event loop which will read messages and dispatch
// them to the InputHandler.
func (c *Conn) Run() error {
	for {
		// Attempt to read the next message
		m, err := c.ReadMessage()
		if err != nil {
			return err
		}

		// The InputHandler can be switched out, but we should
		// never let it be nil.
		if c.InputHandler == nil {
			return errors.New("c.InputHandler is nil")
		}

		// Dispatch the message on the InputHandler
		c.InputHandler.HandleInput(c, m)
	}
}

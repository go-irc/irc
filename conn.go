package irc

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"time"
)

// Conn represents a simple IRC client. It embeds an irc.Reader and an
// irc.Writer.
type Conn struct {
	*Reader
	*Writer
}

// NewConn creates a new Conn
func NewConn(rw io.ReadWriter) *Conn {
	return &Conn{
		NewReader(rw),
		NewWriter(rw),
	}
}

// NewNetConn creates a Conn with optional timeouts
func NewNetConn(conn net.Conn, readTimeout, writeTimeout time.Duration) *Conn {
	return &Conn{
		NewNetReader(conn, readTimeout),
		NewNetWriter(conn, writeTimeout),
	}
}

// Writer is the outgoing side of a connection.
type Writer struct {
	// DebugCallback is called for each outgoing message. The name of this may
	// not be stable.
	DebugCallback func(line string)

	// Internal fields
	writer  io.Writer
	conn    net.Conn
	timeout time.Duration
}

// NewWriter creates an irc.Writer from an io.Writer.
func NewWriter(w io.Writer) *Writer {
	return &Writer{nil, w, nil, 0}
}

// NewNetWriter creates an irc.Writer from a net.Conn and a write timeout.
// Note that the read timeout is not for stream activity but how long waiting
// for a message. These should be almost identical in most situations.
func NewNetWriter(conn net.Conn, timeout time.Duration) *Writer {
	return &Writer{
		nil, conn, conn, timeout,
	}
}

// Write is a simple function which will write the given line to the
// underlying connection.
func (w *Writer) Write(line string) error {
	if w.DebugCallback != nil {
		w.DebugCallback(line)
	}

	if w.conn != nil && w.timeout > 0 {
		err := w.conn.SetWriteDeadline(time.Now().Add(w.timeout))
		if err != nil {
			return err
		}
	}

	_, err := w.writer.Write([]byte(line + "\r\n"))
	return err
}

// Writef is a wrapper around the connection's Write method and
// fmt.Sprintf. Simply use it to send a message as you would normally
// use fmt.Printf.
func (w *Writer) Writef(format string, args ...interface{}) error {
	return w.Write(fmt.Sprintf(format, args...))
}

// WriteMessage writes the given message to the stream
func (w *Writer) WriteMessage(m *Message) error {
	return w.Write(m.String())
}

// Reader is the incoming side of a connection. The data will be
// buffered, so do not re-use the io.Reader used to create the
// irc.Reader.
type Reader struct {
	// DebugCallback is called for each incoming message. The name of this may
	// not be stable.
	DebugCallback func(string)

	// Internal fields
	reader  *bufio.Reader
	conn    net.Conn
	timeout time.Duration
}

// NewReader creates an irc.Reader from an io.Reader.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		nil,
		bufio.NewReader(r),
		nil,
		0,
	}
}

// NewNetReader creates an irc.Reader from a net.Conn and a read timeout. Note
// that the read timeout is not for stream activity but how long waiting for a
// message. These should be almost identical in most situations.
func NewNetReader(c net.Conn, timeout time.Duration) *Reader {
	return &Reader{
		nil,
		bufio.NewReader(c),
		c,
		timeout,
	}
}

// ReadMessage returns the next message from the stream or an error.
func (r *Reader) ReadMessage() (*Message, error) {
	// Set the read deadline if we have one
	if r.conn != nil && r.timeout > 0 {
		err := r.conn.SetReadDeadline(time.Now().Add(r.timeout))
		if err != nil {
			return nil, err
		}
	}

	line, err := r.reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	if r.DebugCallback != nil {
		r.DebugCallback(line)
	}

	// Parse the message from our line
	return ParseMessage(line)
}

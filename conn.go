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

// Writer is the outgoing side of a connection.
type Writer struct {
	// DebugCallback is called for each outgoing message. The name of this may
	// not be stable.
	DebugCallback func(line string)

	// Internal fields
	writer  io.Writer
	timeout time.Duration
}

// NewWriter creates an irc.Writer from an io.Writer.
func NewWriter(w io.Writer) *Writer {
	return &Writer{nil, w, 0}
}

// SetTimeout allows you to set the write timeout for the next call to Write.
// Note that it is undefined behavior to call this while a call to Write is
// happening. Additionally, this is only effective if a net.Conn was passed into
// NewWriter.
func (w *Writer) SetTimeout(timeout time.Duration) {
	w.timeout = timeout
}

// Write is a simple function which will write the given line to the
// underlying connection.
func (w *Writer) Write(line string) error {
	if w.DebugCallback != nil {
		w.DebugCallback(line)
	}

	if c, ok := w.writer.(net.Conn); ok && w.timeout > 0 {
		err := c.SetWriteDeadline(time.Now().Add(w.timeout))
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
	rawReader io.Reader
	reader    *bufio.Reader
	timeout   time.Duration
}

// NewReader creates an irc.Reader from an io.Reader. Note that once a reader is
// passed into this function, you should no longer use it as it is being used
// inside a bufio.Reader so you cannot rely on only the amount of data for a
// Message being read when you call ReadMessage.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		nil,
		r,
		bufio.NewReader(r),
		0,
	}
}

// SetTimeout allows you to set the read timeout for the next call to
// ReadMessage. Note that it is undefined behavior to call this while
// a call to ReadMessage is happening. Additionally, this is only
// effective if a net.Conn is passed into NewReader.
func (r *Reader) SetTimeout(timeout time.Duration) {
	r.timeout = timeout
}

// ReadMessage returns the next message from the stream or an error.
func (r *Reader) ReadMessage() (*Message, error) {
	// Set the read deadline if we have one
	if c, ok := r.rawReader.(net.Conn); ok && r.timeout > 0 {
		err := c.SetReadDeadline(time.Now().Add(r.timeout))
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

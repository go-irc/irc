package irc

import (
	"bufio"
	"fmt"
	"io"
)

// Conn represents a simple IRC client. It embeds an irc.Reader and an
// irc.Writer.
type Conn struct {
	*Reader
	*Writer

	// Internal fields
	closer io.Closer
}

// NewConn creates a new Conn
func NewConn(rw io.ReadWriter) *Conn {
	// Create the client
	c := &Conn{
		NewReader(rw),
		NewWriter(rw),
		nil,
	}

	// If there's a closer available, we want to keep it around
	if closer, ok := rw.(io.Closer); ok {
		c.closer = closer
	}

	return c
}

// Writer is the outgoing side of a connection.
type Writer struct {
	// DebugCallback is called for each outgoing message. The name of this may
	// not be stable.
	DebugCallback func(line string)

	// Internal fields
	writer        io.Writer
	writeCallback func(w *Writer, line string) error
}

func defaultWriteCallback(w *Writer, line string) error {
	_, err := w.writer.Write([]byte(line + "\r\n"))
	return err
}

// NewWriter creates an irc.Writer from an io.Writer.
func NewWriter(w io.Writer) *Writer {
	return &Writer{nil, w, defaultWriteCallback}
}

// Write is a simple function which will write the given line to the
// underlying connection.
func (w *Writer) Write(line string) error {
	if w.DebugCallback != nil {
		w.DebugCallback(line)
	}

	return w.writeCallback(w, line)
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
	reader *bufio.Reader
}

// NewReader creates an irc.Reader from an io.Reader.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		nil,
		bufio.NewReader(r),
	}
}

// ReadMessage returns the next message from the stream or an error.
func (r *Reader) ReadMessage() (*Message, error) {
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

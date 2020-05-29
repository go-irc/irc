package irc

import (
	"bufio"
	"errors"
	"fmt"
	"io"
)

// Conn represents a simple IRC client. It embeds an irc.Reader and an
// irc.Writer.
type Conn struct {
	*Reader
	*Writer
}

// NewConn creates a new Conn.
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

	// WriteCallback is called for each outgoing message. It needs to write the
	// message to the connection. Note that this API is not a part of the semver
	// stability guarantee.
	WriteCallback func(w *Writer, line string) error

	// Internal fields
	writer io.Writer
}

func defaultWriteCallback(w *Writer, line string) error {
	_, err := w.RawWrite([]byte(line + "\r\n"))
	return err
}

// NewWriter creates an irc.Writer from an io.Writer.
func NewWriter(w io.Writer) *Writer {
	return &Writer{nil, defaultWriteCallback, w}
}

// RawWrite will write the given data to the underlying connection, skipping the
// WriteCallback. This is meant to be used by implementations of the
// WriteCallback to write data directly to the stream. Otherwise, it is
// recommended to avoid this function and use one of the other helpers. Also
// note that it will not append \r\n to the end of the line.
func (w *Writer) RawWrite(data []byte) (int, error) {
	return w.writer.Write(data)
}

// Write is a simple function which will write the given line to the
// underlying connection.
func (w *Writer) Write(line string) error {
	if w.DebugCallback != nil {
		w.DebugCallback(line)
	}

	return w.WriteCallback(w, line)
}

// Writef is a wrapper around the connection's Write method and
// fmt.Sprintf. Simply use it to send a message as you would normally
// use fmt.Printf.
func (w *Writer) Writef(format string, args ...interface{}) error {
	return w.Write(fmt.Sprintf(format, args...))
}

// WriteMessage writes the given message to the stream.
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

// NewReader creates an irc.Reader from an io.Reader. Note that once a reader is
// passed into this function, you should no longer use it as it is being used
// inside a bufio.Reader so you cannot rely on only the amount of data for a
// Message being read when you call ReadMessage.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		nil,
		bufio.NewReader(r),
	}
}

// ReadMessage returns the next message from the stream or an error.
// It ignores empty messages.
func (r *Reader) ReadMessage() (*Message, error) {
	var msg *Message

	// It's valid for a message to be empty. Clients should ignore these,
	// so we do to be good citizens.
	err := ErrZeroLengthMessage
	for errors.Is(err, ErrZeroLengthMessage) {
		var line string
		line, err = r.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		if r.DebugCallback != nil {
			r.DebugCallback(line)
		}

		// Parse the message from our line
		msg, err = ParseMessage(line)
	}
	return msg, err
}

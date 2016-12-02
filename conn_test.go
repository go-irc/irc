package irc

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type errorWriter struct{}

func (ew *errorWriter) Write([]byte) (int, error) {
	return 0, errors.New("errorWriter: error")
}

type readWriteCloser struct {
	io.Reader
	io.Writer
	io.Closer
}

type nilCloser struct{}

func (nc *nilCloser) Close() error {
	return nil
}

type testReadWriteCloser struct {
	client *bytes.Buffer
	server *bytes.Buffer
}

func newTestReadWriteCloser() *testReadWriteCloser {
	return &testReadWriteCloser{
		client: &bytes.Buffer{},
		server: &bytes.Buffer{},
	}
}

func (t *testReadWriteCloser) Read(p []byte) (int, error) {
	return t.server.Read(p)
}

func (t *testReadWriteCloser) Write(p []byte) (int, error) {
	return t.client.Write(p)
}

// Ensure we can close the thing
func (t *testReadWriteCloser) Close() error {
	return nil
}

func testReadMessage(t *testing.T, c *Conn) *Message {
	m, err := c.ReadMessage()
	assert.NoError(t, err)
	return m
}

func testLines(t *testing.T, rwc *testReadWriteCloser, expected []string) {
	lines := strings.Split(rwc.client.String(), "\r\n")
	var line, clientLine string
	for len(expected) > 0 {
		line, expected = expected[0], expected[1:]
		clientLine, lines = lines[0], lines[1:]

		assert.Equal(t, line, clientLine)
	}

	for _, line := range lines {
		assert.Equal(t, "", strings.TrimSpace(line), "Extra non-empty lines")
	}

	// Reset the contents
	rwc.client.Reset()
	rwc.server.Reset()
}

func TestWriteMessageError(t *testing.T) {
	t.Parallel()

	rw := readWriteCloser{
		&bytes.Buffer{},
		&errorWriter{},
		&nilCloser{},
	}

	c := NewConn(rw)

	err := c.WriteMessage(MustParseMessage("PING :hello world"))
	assert.Error(t, err)

	err = c.Writef("PING :hello world")
	assert.Error(t, err)

	err = c.Write("PING :hello world")
	assert.Error(t, err)
}

func TestConn(t *testing.T) {
	t.Parallel()

	rwc := newTestReadWriteCloser()
	c := NewConn(rwc)

	// Test writing a message
	m := &Message{Prefix: &Prefix{}, Command: "PING", Params: []string{"Hello World"}}
	c.WriteMessage(m)
	testLines(t, rwc, []string{
		"PING :Hello World",
	})

	// Test with Writef
	c.Writef("PING :%s", "Hello World")
	testLines(t, rwc, []string{
		"PING :Hello World",
	})

	m = MustParseMessage("PONG :Hello World")
	rwc.server.WriteString(m.String() + "\r\n")
	m2 := testReadMessage(t, c)

	assert.EqualValues(t, m, m2, "Message returned by client did not match input")

	// Test welcome message
	rwc.server.WriteString("001 test_nick\r\n")
	m = testReadMessage(t, c)

	rwc.server.WriteString(":invalid_message\r\n")
	_, err := c.ReadMessage()
	assert.Equal(t, ErrMissingDataAfterPrefix, err)

	// This is an odd one... if there wasn't any output, it'll hit
	// EOF, so we expect an error here so we can test an error
	// condition.
	_, err = c.ReadMessage()
	assert.Equal(t, io.EOF, err, "Didn't get expected EOF")
}

package irc_test

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"gopkg.in/irc.v4"
)

var errorWriterErr = errors.New("errorWriter: error")

type errorWriter struct{}

func (ew *errorWriter) Write([]byte) (int, error) {
	return 0, errorWriterErr
}

type nopCloser struct {
	io.Reader
	io.Writer
}

func newNopCloser(inner io.ReadWriter) *nopCloser {
	return &nopCloser{
		Reader: inner,
		Writer: inner,
	}
}

func (nc *nopCloser) Close() error {
	return nil
}

var _ io.ReadWriteCloser = (*nopCloser)(nil)

type readWriteCloser struct {
	io.Reader
	io.Writer
	io.Closer
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

func testReadMessage(t *testing.T, c *irc.Conn) *irc.Message {
	t.Helper()

	m, err := c.ReadMessage()
	assert.NoError(t, err)
	return m
}

func testLines(t *testing.T, rwc *testReadWriteCloser, expected []string) {
	t.Helper()

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

	rw := &readWriteCloser{
		&bytes.Buffer{},
		&errorWriter{},
		nil,
	}

	c := irc.NewConn(rw)

	err := c.WriteMessage(irc.MustParseMessage("PING :hello world"))
	assert.Error(t, err)

	err = c.Writef("PING :hello world")
	assert.Error(t, err)

	err = c.Write("PING :hello world")
	assert.Error(t, err)
}

func TestConn(t *testing.T) {
	t.Parallel()

	rwc := newTestReadWriteCloser()
	c := irc.NewConn(rwc)

	// Test writing a message
	m := &irc.Message{Prefix: &irc.Prefix{}, Command: "PING", Params: []string{"Hello World"}}
	err := c.WriteMessage(m)
	assert.NoError(t, err)
	testLines(t, rwc, []string{
		"PING :Hello World",
	})

	// Test with Writef
	err = c.Writef("PING :%s", "Hello World")
	assert.NoError(t, err)
	testLines(t, rwc, []string{
		"PING :Hello World",
	})

	m = irc.MustParseMessage("PONG :Hello World")
	rwc.server.WriteString(m.String() + "\r\n")
	m2 := testReadMessage(t, c)

	assert.EqualValues(t, m, m2, "Message returned by client did not match input")

	// Test welcome message
	m = irc.MustParseMessage("001 test_nick")
	rwc.server.WriteString(m.String() + "\r\n")
	m2 = testReadMessage(t, c)
	assert.EqualValues(t, m, m2, "Message returned by client did not match input")

	rwc.server.WriteString(":invalid_message\r\n")
	_, err = c.ReadMessage()
	assert.Equal(t, irc.ErrMissingDataAfterPrefix, err)

	// Ensure empty messages are ignored
	m = irc.MustParseMessage("001 test_nick")
	rwc.server.WriteString("\r\n" + m.String() + "\r\n")
	m2 = testReadMessage(t, c)
	assert.EqualValues(t, m, m2, "Message returned by client did not match input")

	// This is an odd one... if there wasn't any output, it'll hit
	// EOF, so we expect an error here so we can test an error
	// condition.
	_, err = c.ReadMessage()
	assert.Equal(t, io.EOF, err, "Didn't get expected EOF")
}

func TestDebugCallback(t *testing.T) {
	t.Parallel()

	var readerHit, writerHit bool
	rwc := newTestReadWriteCloser()
	c := irc.NewConn(rwc)
	c.Writer.DebugCallback = func(string) {
		writerHit = true
	}
	c.Reader.DebugCallback = func(string) {
		readerHit = true
	}

	m := &irc.Message{Prefix: &irc.Prefix{}, Command: "PING", Params: []string{"Hello World"}}
	err := c.WriteMessage(m)
	assert.NoError(t, err)
	testLines(t, rwc, []string{
		"PING :Hello World",
	})
	m = irc.MustParseMessage("PONG :Hello World")
	rwc.server.WriteString(m.String() + "\r\n")
	testReadMessage(t, c)

	assert.True(t, readerHit)
	assert.True(t, writerHit)
}

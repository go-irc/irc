package irc

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

	// Ensure CTCP messages are parsed
	rwc.server.WriteString(":world PRIVMSG :\x01VERSION\x01\r\n")
	m = testReadMessage(t, c)
	assert.Equal(t, "CTCP", m.Command, "Message was not parsed as CTCP")
	assert.Equal(t, "VERSION", m.Trailing(), "Wrong CTCP command")

	rwc.server.WriteString(":invalid_message\r\n")
	_, err := c.ReadMessage()
	assert.Equal(t, ErrMissingDataAfterPrefix, err)

	// This is an odd one... if there wasn't any output, it'll hit
	// EOF, so we expect an error here so we can test an error
	// condition.
	_, err = c.ReadMessage()
	assert.Equal(t, io.EOF, err, "Didn't get expected EOF")
}

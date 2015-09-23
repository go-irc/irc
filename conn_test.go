package irc

import (
	"bytes"
	"io"
	"reflect"
	"strings"
	"testing"
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
	if err != nil {
		t.Error(err)
	}
	return m
}

func testLines(t *testing.T, rwc *testReadWriteCloser, expected []string) {
	lines := strings.Split(rwc.client.String(), "\r\n")
	var line, clientLine string
	for len(expected) > 0 {
		line, expected = expected[0], expected[1:]
		clientLine, lines = lines[0], lines[1:]

		if line != clientLine {
			t.Errorf("Expected %s != Got %s", line, clientLine)
		}
	}

	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			t.Errorf("Extra non-empty lines: %s", line)
		}
	}

	// Reset the contents
	rwc.client.Reset()
	rwc.server.Reset()
}

func TestClient(t *testing.T) {
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

	m = ParseMessage("PONG :Hello World")
	rwc.server.WriteString(m.String() + "\r\n")
	m2 := testReadMessage(t, c)

	if !reflect.DeepEqual(m, m2) {
		t.Errorf("Message returned by client did not match input")
	}

	// Test welcome message
	rwc.server.WriteString("001 test_nick\r\n")
	m = testReadMessage(t, c)

	// Ensure CTCP messages are parsed
	rwc.server.WriteString(":world PRIVMSG :\x01VERSION\x01\r\n")
	m = testReadMessage(t, c)
	if m.Command != "CTCP" {
		t.Error("Message was not parsed as CTCP")
	}
	if m.Trailing() != "VERSION" {
		t.Error("Wrong CTCP command")
	}

	// This is an odd one... if there wasn't any output, it'll hit
	// EOF, so we expect an error here so we can test an error
	// condition.
	_, err := c.ReadMessage()
	if err != io.EOF {
		t.Error("Didn't get expected EOF error")
	}
}

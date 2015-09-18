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

func testReadMessage(t *testing.T, c *Client) *Message {
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
			t.Errorf("Extra non-empty lines")
		}
	}

	// Reset the contents
	rwc.client.Reset()
	rwc.server.Reset()
}

func TestClient(t *testing.T) {
	rwc := newTestReadWriteCloser()
	c := NewClient(rwc, "test_nick", "test_user", "test_name", "test_pass")

	testLines(t, rwc, []string{
		"PASS test_pass",
		"NICK test_nick",
		"USER test_user 0.0.0.0 0.0.0.0 :test_name",
	})

	if c.CurrentNick() != "test_nick" {
		t.Errorf("c.CurrentNick was %s, not test_nick", c.CurrentNick())
	}

	// Test writing a message
	m := &Message{Prefix: &Prefix{}, Command: "PING", Params: []string{"Hello World"}}
	c.WriteMessage(m)

	testLines(t, rwc, []string{
		"PING :Hello World",
	})

	m = ParseMessage("PONG :Hello World")
	rwc.server.WriteString(m.String() + "\r\n")
	m2 := testReadMessage(t, c)

	if !reflect.DeepEqual(m, m2) {
		t.Errorf("Message returned by client did not match input")
	}

	// Test specific messages
	rwc.server.WriteString("PING :42\r\n")
	m = testReadMessage(t, c)

	testLines(t, rwc, []string{
		"PONG :42",
	})

	// Test nick change
	rwc.server.WriteString(":test_nick NICK new_test_nick\r\n")
	m = testReadMessage(t, c)

	if c.CurrentNick() != "new_test_nick" {
		t.Errorf("c.CurrentNick was %s, not new_test_nick", c.CurrentNick())
	}

	// Test welcome message
	rwc.server.WriteString("001 test_nick\r\n")
	m = testReadMessage(t, c)

	if c.CurrentNick() != "test_nick" {
		t.Errorf("c.CurrentNick was %s, not test_nick", c.CurrentNick())
	}

	// Test nick collisions
	rwc.server.WriteString("437\r\n")
	m = testReadMessage(t, c)
	testLines(t, rwc, []string{
		"NICK test_nick_",
	})

	// Ensure CTCP messages are parsed
	rwc.server.WriteString(":world PRIVMSG :\x01VERSION\x01\r\n")
	m = testReadMessage(t, c)
	if m.Command != "CTCP" {
		t.Error("Message was not parsed as CTCP")
	}
	if m.Trailing() != "VERSION" {
		t.Error("Wrong CTCP command")
	}

	// Test CTCPReply
	c.CTCPReply(m, "VERSION 42")
	testLines(t, rwc, []string{
		"NOTICE world :\x01VERSION 42\x01",
	})

	// This is an odd one... if there wasn't any output, it'll hit
	// EOF, so we expect an error here so we can test an error
	// condition.
	_, err := c.ReadMessage()
	if err != io.EOF {
		t.Error("Didn't get expected EOF error")
	}

	mInvalid := &Message{}
	mFromUser := &Message{
		Prefix:  &Prefix{Name: "seabot"},
		Command: "PRIVMSG",
		Params:  []string{"seabot", "Hello"},
	}
	mFromChannel := &Message{
		Prefix:  &Prefix{Name: "seabot"},
		Command: "PRIVMSG",
		Params:  []string{"#seabot", "Hello"},
	}

	c.MentionReply(mFromUser, "hi")
	c.MentionReply(mFromChannel, "hi")
	testLines(t, rwc, []string{
		"PRIVMSG seabot :hi",
		"PRIVMSG #seabot :seabot: hi",
	})

	if c.Reply(mInvalid, "TEST") == nil {
		t.Error("Expected error, didn't get one")
	}
	if c.MentionReply(mInvalid, "TEST") == nil {
		t.Error("Expected error, didn't get one")
	}
	if c.CTCPReply(mInvalid, "TEST") == nil {
		t.Errorf("Expected error, didn't get one")
	}
}

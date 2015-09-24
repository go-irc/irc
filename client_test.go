package irc

import (
	"io"
	"testing"
)

func TestClient(t *testing.T) {
	hit := false
	rwc := newTestReadWriteCloser()
	c := NewClient(rwc, nil)
	c.Handler = HandlerFunc(func(c *Client, m *Message) {
		hit = true
	})
	err := c.Run("test_nick", "test_user", "test_name", "pass")
	if err != io.EOF {
		t.Error(err)
	}

	testLines(t, rwc, []string{
		"PASS :pass",
		"NICK :test_nick",
		"USER test_user 0.0.0.0 0.0.0.0 :test_name",
	})

	if c.CurrentNick() != "test_nick" {
		t.Errorf("CurrentNick (%s) != test_nick", c.CurrentNick())
	}

	// Now that we have what's written to the stream out of the
	// way, we can start testing the handling of certain commands.
	rwc.server.WriteString("PING :hello world\r\n")
	c.Run("test_nick", "test_user", "test_name", "pass")
	testLines(t, rwc, []string{
		"PASS :pass",
		"NICK :test_nick",
		"USER test_user 0.0.0.0 0.0.0.0 :test_name",
		"PONG :hello world",
	})

	rwc.server.WriteString(":test_nick NICK :test_nick_2\r\n")
	c.Run("test_nick", "test_user", "test_name", "pass")
	testLines(t, rwc, []string{
		"PASS :pass",
		"NICK :test_nick",
		"USER test_user 0.0.0.0 0.0.0.0 :test_name",
	})
	if c.CurrentNick() != "test_nick_2" {
		t.Errorf("CurrentNick (%s) != test_nick_2", c.CurrentNick())
	}

	rwc.server.WriteString("001 :test_nick\r\n")
	c.Run("test_nick", "test_user", "test_name", "pass")
	testLines(t, rwc, []string{
		"PASS :pass",
		"NICK :test_nick",
		"USER test_user 0.0.0.0 0.0.0.0 :test_name",
	})
	if c.CurrentNick() != "test_nick" {
		t.Errorf("CurrentNick (%s) != test_nick", c.CurrentNick())
	}

	rwc.server.WriteString("433\r\n")
	c.Run("test_nick", "test_user", "test_name", "pass")
	testLines(t, rwc, []string{
		"PASS :pass",
		"NICK :test_nick",
		"USER test_user 0.0.0.0 0.0.0.0 :test_name",
		"NICK :test_nick_",
	})
	if c.CurrentNick() != "test_nick_" {
		t.Errorf("CurrentNick (%s) != test_nick", c.CurrentNick())
	}

	if !hit {
		t.Errorf("Handler was not hit")
	}
}

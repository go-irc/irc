package irc

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestHandler struct {
	messages []*Message
}

func (th *TestHandler) Handle(c *Client, m *Message) {
	th.messages = append(th.messages, m)
}

func (th *TestHandler) Messages() []*Message {
	ret := th.messages
	th.messages = nil
	return ret
}

func TestClient(t *testing.T) {
	t.Parallel()

	rwc := newTestReadWriteCloser()
	config := ClientConfig{
		Nick: "test_nick",
		Pass: "test_pass",
		User: "test_user",
		Name: "test_name",
	}
	c := NewClient(rwc, config)
	err := c.Run()
	assert.Equal(t, io.EOF, err)

	testLines(t, rwc, []string{
		"PASS :test_pass",
		"NICK :test_nick",
		"USER test_user 0.0.0.0 0.0.0.0 :test_name",
	})

	rwc.server.WriteString("PING :hello world\r\n")
	err = c.Run()
	assert.Equal(t, io.EOF, err)
	testLines(t, rwc, []string{
		"PASS :test_pass",
		"NICK :test_nick",
		"USER test_user 0.0.0.0 0.0.0.0 :test_name",
		"PONG :hello world",
	})

	rwc.server.WriteString(":test_nick NICK :new_test_nick\r\n")
	err = c.Run()
	assert.Equal(t, io.EOF, err)
	testLines(t, rwc, []string{
		"PASS :test_pass",
		"NICK :test_nick",
		"USER test_user 0.0.0.0 0.0.0.0 :test_name",
	})
	assert.Equal(t, "new_test_nick", c.CurrentNick())

	rwc.server.WriteString("001 :new_test_nick\r\n")
	err = c.Run()
	assert.Equal(t, io.EOF, err)
	testLines(t, rwc, []string{
		"PASS :test_pass",
		"NICK :test_nick",
		"USER test_user 0.0.0.0 0.0.0.0 :test_name",
	})
	assert.Equal(t, "new_test_nick", c.CurrentNick())

	rwc.server.WriteString("433\r\n")
	err = c.Run()
	assert.Equal(t, io.EOF, err)
	testLines(t, rwc, []string{
		"PASS :test_pass",
		"NICK :test_nick",
		"USER test_user 0.0.0.0 0.0.0.0 :test_name",
		"NICK :test_nick_",
	})
	assert.Equal(t, "test_nick_", c.CurrentNick())
}

func TestClientHandler(t *testing.T) {
	t.Parallel()

	handler := &TestHandler{}
	rwc := newTestReadWriteCloser()
	config := ClientConfig{
		Nick: "test_nick",
		Pass: "test_pass",
		User: "test_user",
		Name: "test_name",

		Handler: handler,
	}

	rwc.server.WriteString("001 :hello_world\r\n")
	c := NewClient(rwc, config)
	err := c.Run()
	assert.Equal(t, io.EOF, err)

	testLines(t, rwc, []string{
		"PASS :test_pass",
		"NICK :test_nick",
		"USER test_user 0.0.0.0 0.0.0.0 :test_name",
	})

	assert.EqualValues(t, []*Message{
		&Message{
			Prefix:  &Prefix{},
			Command: "001",
			Params:  []string{"hello_world"},
		},
	}, handler.Messages())
}

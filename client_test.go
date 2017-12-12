package irc

import (
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type TestHandler struct {
	messages []*Message
	delay    time.Duration
}

func (th *TestHandler) Handle(c *Client, m *Message) {
	th.messages = append(th.messages, m)
	if th.delay > 0 {
		time.Sleep(th.delay)
	}
}

func (th *TestHandler) Messages() []*Message {
	ret := th.messages
	th.messages = nil
	return ret
}

func TestCapReq(t *testing.T) {
	t.Parallel()

	config := ClientConfig{
		Nick: "test_nick",
		Pass: "test_pass",
		User: "test_user",
		Name: "test_name",
	}

	c := runClientTest(t, config, io.EOF, func(c *Client) {
		assert.False(t, c.CapAvailable("random-thing"))
		assert.False(t, c.CapAvailable("multi-prefix"))
		c.CapRequest("multi-prefix", true)
	}, []TestAction{
		ExpectLine("PASS :test_pass\r\n"),
		ExpectLine("CAP LS\r\n"),
		ExpectLine("CAP REQ :multi-prefix\r\n"),
		SendLine("CAP * LS :multi-prefix\r\n"),
		SendLine("CAP * ACK :multi-prefix\r\n"),
		ExpectLine("CAP END\r\n"),
		ExpectLine("NICK :test_nick\r\n"),
		ExpectLine("USER test_user 0.0.0.0 0.0.0.0 :test_name\r\n"),
	})
	assert.False(t, c.CapEnabled("random-thing"))
	assert.True(t, c.CapEnabled("multi-prefix"))
	assert.False(t, c.CapAvailable("random-thing"))
	assert.True(t, c.CapAvailable("multi-prefix"))

	// Malformed CAP responses are ignored
	c = runClientTest(t, config, io.EOF, func(c *Client) {
		assert.False(t, c.CapAvailable("random-thing"))
		assert.False(t, c.CapAvailable("multi-prefix"))
		c.CapRequest("multi-prefix", true)
	}, []TestAction{
		ExpectLine("PASS :test_pass\r\n"),
		ExpectLine("CAP LS\r\n"),
		ExpectLine("CAP REQ :multi-prefix\r\n"),
		SendLine("CAP * LS :multi-prefix\r\n"),
		//SendLine("CAP * ACK\r\n"), // Malformed CAP response
		SendLine("CAP * ACK :multi-prefix\r\n"),
		ExpectLine("CAP END\r\n"),
		ExpectLine("NICK :test_nick\r\n"),
		ExpectLine("USER test_user 0.0.0.0 0.0.0.0 :test_name\r\n"),
	})
	assert.False(t, c.CapEnabled("random-thing"))
	assert.True(t, c.CapEnabled("multi-prefix"))
	assert.False(t, c.CapAvailable("random-thing"))
	assert.True(t, c.CapAvailable("multi-prefix"))

	// Additional CAP messages after the start are ignored.
	c = runClientTest(t, config, io.EOF, func(c *Client) {
		assert.False(t, c.CapAvailable("random-thing"))
		assert.False(t, c.CapAvailable("multi-prefix"))
		c.CapRequest("multi-prefix", true)
	}, []TestAction{
		ExpectLine("PASS :test_pass\r\n"),
		ExpectLine("CAP LS\r\n"),
		ExpectLine("CAP REQ :multi-prefix\r\n"),
		SendLine("CAP * LS :multi-prefix\r\n"),
		SendLine("CAP * ACK :multi-prefix\r\n"),
		ExpectLine("CAP END\r\n"),
		ExpectLine("NICK :test_nick\r\n"),
		ExpectLine("USER test_user 0.0.0.0 0.0.0.0 :test_name\r\n"),
		SendLine("CAP * NAK :multi-prefix\r\n"),
	})
	assert.False(t, c.CapEnabled("random-thing"))
	assert.True(t, c.CapEnabled("multi-prefix"))
	assert.False(t, c.CapAvailable("random-thing"))
	assert.True(t, c.CapAvailable("multi-prefix"))

	c = runClientTest(t, config, io.EOF, func(c *Client) {
		assert.False(t, c.CapAvailable("random-thing"))
		assert.False(t, c.CapAvailable("multi-prefix"))
		c.CapRequest("multi-prefix", false)
	}, []TestAction{
		ExpectLine("PASS :test_pass\r\n"),
		ExpectLine("CAP LS\r\n"),
		ExpectLine("CAP REQ :multi-prefix\r\n"),
		SendLine("CAP * LS :multi-prefix\r\n"),
		SendLine("CAP * NAK :multi-prefix\r\n"),
		ExpectLine("CAP END\r\n"),
		ExpectLine("NICK :test_nick\r\n"),
		ExpectLine("USER test_user 0.0.0.0 0.0.0.0 :test_name\r\n"),
	})
	assert.False(t, c.CapEnabled("random-thing"))
	assert.False(t, c.CapEnabled("multi-prefix"))
	assert.False(t, c.CapAvailable("random-thing"))
	assert.True(t, c.CapAvailable("multi-prefix"))

	c = runClientTest(t, config, errors.New("CAP multi-prefix requested but was rejected"), func(c *Client) {
		assert.False(t, c.CapAvailable("random-thing"))
		assert.False(t, c.CapAvailable("multi-prefix"))
		c.CapRequest("multi-prefix", true)
	}, []TestAction{
		ExpectLine("PASS :test_pass\r\n"),
		ExpectLine("CAP LS\r\n"),
		ExpectLine("CAP REQ :multi-prefix\r\n"),
		SendLine("CAP * LS :multi-prefix\r\n"),
		SendLine("CAP * NAK :multi-prefix\r\n"),
	})
	assert.False(t, c.CapEnabled("random-thing"))
	assert.False(t, c.CapEnabled("multi-prefix"))
	assert.False(t, c.CapAvailable("random-thing"))
	assert.True(t, c.CapAvailable("multi-prefix"))

	c = runClientTest(t, config, errors.New("CAP multi-prefix requested but not accepted"), func(c *Client) {
		assert.False(t, c.CapAvailable("random-thing"))
		assert.False(t, c.CapAvailable("multi-prefix"))
		c.CapRequest("multi-prefix", true)
	}, []TestAction{
		ExpectLine("PASS :test_pass\r\n"),
		ExpectLine("CAP LS\r\n"),
		ExpectLine("CAP REQ :multi-prefix\r\n"),
		SendLine("CAP * LS :multi-prefix\r\n"),
		SendLine("CAP * ACK :\r\n"),
	})
	assert.False(t, c.CapEnabled("random-thing"))
	assert.False(t, c.CapEnabled("multi-prefix"))
	assert.False(t, c.CapAvailable("random-thing"))
	assert.True(t, c.CapAvailable("multi-prefix"))
}

func TestClient(t *testing.T) {
	t.Parallel()

	config := ClientConfig{
		Nick: "test_nick",
		Pass: "test_pass",
		User: "test_user",
		Name: "test_name",
	}

	runClientTest(t, config, io.EOF, nil, []TestAction{
		ExpectLine("PASS :test_pass\r\n"),
		ExpectLine("CAP LS\r\n"),
		SendLine("CAP * LS :\r\n"),
		ExpectLine("CAP END\r\n"),
		ExpectLine("NICK :test_nick\r\n"),
		ExpectLine("USER test_user 0.0.0.0 0.0.0.0 :test_name\r\n"),
	})

	runClientTest(t, config, io.EOF, nil, []TestAction{
		ExpectLine("PASS :test_pass\r\n"),
		ExpectLine("CAP LS\r\n"),
		SendLine("CAP * LS :\r\n"),
		ExpectLine("CAP END\r\n"),
		ExpectLine("NICK :test_nick\r\n"),
		ExpectLine("USER test_user 0.0.0.0 0.0.0.0 :test_name\r\n"),
		SendLine("PING :hello world\r\n"),
		ExpectLine("PONG :hello world\r\n"),
	})

	c := runClientTest(t, config, io.EOF, nil, []TestAction{
		ExpectLine("PASS :test_pass\r\n"),
		ExpectLine("CAP LS\r\n"),
		SendLine("CAP * LS :\r\n"),
		ExpectLine("CAP END\r\n"),
		ExpectLine("NICK :test_nick\r\n"),
		ExpectLine("USER test_user 0.0.0.0 0.0.0.0 :test_name\r\n"),
		SendLine(":test_nick NICK :new_test_nick\r\n"),
	})
	assert.Equal(t, "new_test_nick", c.CurrentNick())

	c = runClientTest(t, config, io.EOF, nil, []TestAction{
		ExpectLine("PASS :test_pass\r\n"),
		ExpectLine("CAP LS\r\n"),
		SendLine("CAP * LS :\r\n"),
		ExpectLine("CAP END\r\n"),
		ExpectLine("NICK :test_nick\r\n"),
		ExpectLine("USER test_user 0.0.0.0 0.0.0.0 :test_name\r\n"),
		SendLine("001 :new_test_nick\r\n"),
	})
	assert.Equal(t, "new_test_nick", c.CurrentNick())

	c = runClientTest(t, config, io.EOF, nil, []TestAction{
		ExpectLine("PASS :test_pass\r\n"),
		ExpectLine("CAP LS\r\n"),
		SendLine("CAP * LS :\r\n"),
		ExpectLine("CAP END\r\n"),
		ExpectLine("NICK :test_nick\r\n"),
		ExpectLine("USER test_user 0.0.0.0 0.0.0.0 :test_name\r\n"),
		SendLine("433\r\n"),
		ExpectLine("NICK :test_nick_\r\n"),
	})
	assert.Equal(t, "test_nick_", c.CurrentNick())

	c = runClientTest(t, config, io.EOF, nil, []TestAction{
		ExpectLine("PASS :test_pass\r\n"),
		ExpectLine("CAP LS\r\n"),
		SendLine("CAP * LS :\r\n"),
		ExpectLine("CAP END\r\n"),
		ExpectLine("NICK :test_nick\r\n"),
		ExpectLine("USER test_user 0.0.0.0 0.0.0.0 :test_name\r\n"),
		SendLine("437\r\n"),
		ExpectLine("NICK :test_nick_\r\n"),
	})
	assert.Equal(t, "test_nick_", c.CurrentNick())
}

func TestSendLimit(t *testing.T) {
	t.Parallel()

	handler := &TestHandler{}

	config := ClientConfig{
		Nick: "test_nick",
		Pass: "test_pass",
		User: "test_user",
		Name: "test_name",

		Handler: handler,

		SendLimit: 10 * time.Millisecond,
		SendBurst: 2,
	}

	before := time.Now()
	runClientTest(t, config, io.EOF, nil, []TestAction{
		ExpectLine("PASS :test_pass\r\n"),
		ExpectLine("CAP LS\r\n"),
		SendLine("CAP * LS :\r\n"),
		ExpectLine("CAP END\r\n"),
		ExpectLine("NICK :test_nick\r\n"),
		ExpectLine("USER test_user 0.0.0.0 0.0.0.0 :test_name\r\n"),
		SendLine("001 :hello_world\r\n"),
	})
	assert.WithinDuration(t, before, time.Now(), 60*time.Millisecond)

	// This last test isn't really a test. It's being used to make sure we
	// hit the branch which handles dropping ticks if the buffered channel is
	// full.
	handler.delay = 20 * time.Millisecond // Sleep for 20ms when we get the 001 message
	config.SendLimit = 10 * time.Millisecond
	config.SendBurst = 0

	before = time.Now()
	runClientTest(t, config, io.EOF, nil, []TestAction{
		ExpectLine("PASS :test_pass\r\n"),
		ExpectLine("CAP LS\r\n"),
		SendLine("CAP * LS :\r\n"),
		ExpectLine("CAP END\r\n"),
		ExpectLine("NICK :test_nick\r\n"),
		ExpectLine("USER test_user 0.0.0.0 0.0.0.0 :test_name\r\n"),
		SendLine("001 :hello_world\r\n"),
	})
	assert.WithinDuration(t, before, time.Now(), 80*time.Millisecond)
}

func TestClientHandler(t *testing.T) {
	t.Parallel()

	handler := &TestHandler{}
	config := ClientConfig{
		Nick: "test_nick",
		Pass: "test_pass",
		User: "test_user",
		Name: "test_name",

		Handler: handler,
	}

	runClientTest(t, config, io.EOF, nil, []TestAction{
		ExpectLine("PASS :test_pass\r\n"),
		ExpectLine("CAP LS\r\n"),
		SendLine("CAP * LS :\r\n"),
		ExpectLine("CAP END\r\n"),
		ExpectLine("NICK :test_nick\r\n"),
		ExpectLine("USER test_user 0.0.0.0 0.0.0.0 :test_name\r\n"),
		SendLine("001 :hello_world\r\n"),
	})
	assert.EqualValues(t, []*Message{
		{
			Tags:    Tags{},
			Prefix:  &Prefix{},
			Command: "001",
			Params:  []string{"hello_world"},
		},
	}, handler.Messages())

	// Ensure CTCP messages are parsed
	runClientTest(t, config, io.EOF, nil, []TestAction{
		ExpectLine("PASS :test_pass\r\n"),
		ExpectLine("CAP LS\r\n"),
		SendLine("CAP * LS :\r\n"),
		ExpectLine("CAP END\r\n"),
		ExpectLine("NICK :test_nick\r\n"),
		ExpectLine("USER test_user 0.0.0.0 0.0.0.0 :test_name\r\n"),
		SendLine(":world PRIVMSG :\x01VERSION\x01\r\n"),
	})
	assert.EqualValues(t, []*Message{
		{
			Tags:    Tags{},
			Prefix:  &Prefix{Name: "world"},
			Command: "CTCP",
			Params:  []string{"VERSION"},
		},
	}, handler.Messages())

	// CTCP Regression test for PR#47
	// Proper CTCP should start AND end in \x01
	runClientTest(t, config, io.EOF, nil, []TestAction{
		ExpectLine("PASS :test_pass\r\n"),
		ExpectLine("CAP LS\r\n"),
		SendLine("CAP * LS :\r\n"),
		ExpectLine("CAP END\r\n"),
		ExpectLine("NICK :test_nick\r\n"),
		ExpectLine("USER test_user 0.0.0.0 0.0.0.0 :test_name\r\n"),
		SendLine(":world PRIVMSG :\x01VERSION\r\n"),
	})
	assert.EqualValues(t, []*Message{
		{
			Tags:    Tags{},
			Prefix:  &Prefix{Name: "world"},
			Command: "PRIVMSG",
			Params:  []string{"\x01VERSION"},
		},
	}, handler.Messages())
}

func TestFromChannel(t *testing.T) {
	t.Parallel()

	c := Client{currentNick: "test_nick"}
	m := MustParseMessage("PRIVMSG test_nick :hello world")
	assert.False(t, c.FromChannel(m))

	m = MustParseMessage("PRIVMSG #a_channel :hello world")
	assert.True(t, c.FromChannel(m))

	m = MustParseMessage("PING")
	assert.False(t, c.FromChannel(m))
}

func TestPingLoop(t *testing.T) {
	t.Parallel()

	config := ClientConfig{
		Nick: "test_nick",
		Pass: "test_pass",
		User: "test_user",
		Name: "test_name",

		PingFrequency: 20 * time.Millisecond,
		PingTimeout:   5 * time.Millisecond,
	}

	var lastPing *Message

	// Successful ping
	runClientTest(t, config, io.EOF, nil, []TestAction{
		ExpectLine("PASS :test_pass\r\n"),
		ExpectLine("CAP LS\r\n"),
		SendLine("CAP * LS :\r\n"),
		ExpectLine("CAP END\r\n"),
		ExpectLine("NICK :test_nick\r\n"),
		ExpectLine("USER test_user 0.0.0.0 0.0.0.0 :test_name\r\n"),
		SendLine("001 :hello_world\r\n"),
		Delay(20 * time.Millisecond),
		LineFunc(func(m *Message) {
			lastPing = m
		}),
		SendFunc(func() string {
			return fmt.Sprintf("PONG :%s\r\n", lastPing.Trailing())
		}),
		Delay(10 * time.Millisecond),
	})

	// Ping timeout
	runClientTest(t, config, errors.New("Ping Timeout"), nil, []TestAction{
		ExpectLine("PASS :test_pass\r\n"),
		ExpectLine("CAP LS\r\n"),
		SendLine("CAP * LS :\r\n"),
		ExpectLine("CAP END\r\n"),
		ExpectLine("NICK :test_nick\r\n"),
		ExpectLine("USER test_user 0.0.0.0 0.0.0.0 :test_name\r\n"),
		SendLine("001 :hello_world\r\n"),
		Delay(20 * time.Millisecond),
		LineFunc(func(m *Message) {
			lastPing = m
		}),
		Delay(20 * time.Millisecond),
	})

	// Exit in the middle of handling a ping
	runClientTest(t, config, io.EOF, nil, []TestAction{
		ExpectLine("PASS :test_pass\r\n"),
		ExpectLine("CAP LS\r\n"),
		SendLine("CAP * LS :\r\n"),
		ExpectLine("CAP END\r\n"),
		ExpectLine("NICK :test_nick\r\n"),
		ExpectLine("USER test_user 0.0.0.0 0.0.0.0 :test_name\r\n"),
		SendLine("001 :hello_world\r\n"),
		Delay(20 * time.Millisecond),
		LineFunc(func(m *Message) {
			lastPing = m
		}),
	})

	// This one is just for coverage, so we know we're hitting the
	// branch that drops extra pings.
	runClientTest(t, config, io.EOF, nil, []TestAction{
		ExpectLine("PASS :test_pass\r\n"),
		ExpectLine("CAP LS\r\n"),
		SendLine("CAP * LS :\r\n"),
		ExpectLine("CAP END\r\n"),
		ExpectLine("NICK :test_nick\r\n"),
		ExpectLine("USER test_user 0.0.0.0 0.0.0.0 :test_name\r\n"),
		SendLine("001 :hello_world\r\n"),

		// It's a buffered channel of 5, so we want to send at least 6 of them
		SendLine("PONG :hello 1\r\n"),
		SendLine("PONG :hello 2\r\n"),
		SendLine("PONG :hello 3\r\n"),
		SendLine("PONG :hello 4\r\n"),
		SendLine("PONG :hello 5\r\n"),
		SendLine("PONG :hello 6\r\n"),
		SendLine("PONG :hello 7\r\n"),
	})
}

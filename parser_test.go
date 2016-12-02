package irc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var messageTests = []struct {
	// Message parsing
	Prefix, Cmd string
	Params      []string

	// Tag parsing
	Tags Tags

	// Prefix parsing
	Name, User, Host string

	// Total output
	Expect   string
	ExpectIn []string
	Err      string

	// FromChannel
	FromChan bool
}{
	{
		Err: "Cannot parse zero-length message",
	},
	{
		Expect: ":asd  :",
		Err:    "Missing message command",
	},
	{
		Expect: ":A",
		Err:    "No message data after prefix",
	},
	{
		Expect: "@A",
		Err:    "No message data after tags",
	},
	{
		Prefix: "server.kevlar.net",
		Cmd:    "PING",
		Params: []string{},

		Name: "server.kevlar.net",

		Expect: ":server.kevlar.net PING\n",
	},
	{
		Prefix: "server.kevlar.net",
		Cmd:    "NOTICE",
		Params: []string{"user", "*** This is a test"},

		Name: "server.kevlar.net",

		Expect: ":server.kevlar.net NOTICE user :*** This is a test\n",
	},
	{
		Prefix: "belakA!belakB@a.host.com",
		Cmd:    "PRIVMSG",
		Params: []string{"#somewhere", "*** This is a test"},

		Name: "belakA",
		User: "belakB",
		Host: "a.host.com",

		Expect:   ":belakA!belakB@a.host.com PRIVMSG #somewhere :*** This is a test\n",
		FromChan: true,
	},
	{
		Prefix: "freenode",
		Cmd:    "005",
		Params: []string{"starkbot", "CHANLIMIT=#:120", "MORE", "are supported by this server"},

		Name: "freenode",

		Expect: ":freenode 005 starkbot CHANLIMIT=#:120 MORE :are supported by this server\n",
	},
	{
		Prefix: "belakA!belakB@a.host.com",
		Cmd:    "PRIVMSG",
		Params: []string{"&somewhere", "*** This is a test"},

		Name: "belakA",
		User: "belakB",
		Host: "a.host.com",

		Expect:   ":belakA!belakB@a.host.com PRIVMSG &somewhere :*** This is a test\n",
		FromChan: true,
	},
	{
		Prefix: "belakA!belakB@a.host.com",
		Cmd:    "PRIVMSG",
		Params: []string{"belak", "*** This is a test"},

		Name: "belakA",
		User: "belakB",
		Host: "a.host.com",

		Expect: ":belakA!belakB@a.host.com PRIVMSG belak :*** This is a test\n",
	},
	{
		Prefix: "A",
		Cmd:    "B",
		Params: []string{"C"},

		Name: "A",

		Expect: ":A B C\n",
	},
	{
		Prefix: "A@B",
		Cmd:    "C",
		Params: []string{"D"},

		Name: "A",
		Host: "B",

		Expect: ":A@B C D\n",
	},
	{
		Cmd:    "B",
		Params: []string{"C"},
		Expect: "B C\n",
	},
	{
		Prefix: "A",
		Cmd:    "B",
		Params: []string{"C", "D"},

		Name: "A",

		Expect: ":A B C D\n",
	},
	{
		Cmd:    "A",
		Params: []string{""},

		Expect: "A :\n",
	},
	{
		Tags: Tags{
			"tag": "value",
		},

		Params: []string{},
		Cmd:    "A",

		Expect: "@tag=value A\n",
	},
	{
		Tags: Tags{
			"tag": "\n",
		},

		Params: []string{},
		Cmd:    "A",

		Expect: "@tag=\\n A\n",
	},
	{
		Tags: Tags{
			"tag": "\\",
		},

		Params: []string{},
		Cmd:    "A",

		Expect:   "@tag=\\ A\n",
		ExpectIn: []string{"@tag=\\\\ A\n"},
	},
	{
		Tags: Tags{
			"tag": ";",
		},

		Params: []string{},
		Cmd:    "A",

		Expect: "@tag=\\: A\n",
	},
	{
		Tags: Tags{
			"tag": "",
		},

		Params: []string{},
		Cmd:    "A",

		Expect: "@tag A\n",
	},
	{
		Tags: Tags{
			"tag": "\\&",
		},

		Params: []string{},
		Cmd:    "A",

		Expect:   "@tag=\\& A\n",
		ExpectIn: []string{"@tag=\\\\& A\n"},
	},
	{
		Tags: Tags{
			"tag":  "x",
			"tag2": "asd",
		},

		Params: []string{},
		Cmd:    "A",

		Expect:   "@tag=x;tag2=asd A\n",
		ExpectIn: []string{"@tag=x;tag2=asd A\n", "@tag2=asd;tag=x A\n"},
	},
	{
		Tags: Tags{
			"tag": "; \\\r\n",
		},

		Params: []string{},
		Cmd:    "A",
		Expect: "@tag=\\:\\s\\\\\\r\\n A\n",
	},
}

func TestMustParseMessage(t *testing.T) {
	t.Parallel()

	for i, test := range messageTests {
		if test.Err != "" {
			assert.Panics(t, func() {
				MustParseMessage(test.Expect)
			}, "%d. Didn't get expected panic", i)
		} else {
			assert.NotPanics(t, func() {
				MustParseMessage(test.Expect)
			}, "%d. Got unexpected panic", i)
		}
	}
}

func TestParseMessage(t *testing.T) {
	t.Parallel()

	for i, test := range messageTests {
		m, err := ParseMessage(test.Expect)
		if test.Err != "" {
			assert.Equal(t, test.Err, err.Error(), "%d. Didn't get correct error for invalid message.", i)
		} else {
			assert.NotNil(t, m, "%d. Got nil for valid message.", i)
		}

		if m == nil {
			continue
		}

		assert.Equal(t, test.Cmd, m.Command, "%d. Command doesn't match.", i)
		assert.EqualValues(t, test.Params, m.Params, "%d. Params don't match.", i)
	}
}

func BenchmarkParseMessage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseMessage(messageTests[i%len(messageTests)].Prefix)
	}
}

func TestParsePrefix(t *testing.T) {
	t.Parallel()

	for i, test := range messageTests {
		pi := ParsePrefix(test.Prefix)
		if pi == nil {
			t.Errorf("%d. Got nil for valid identity", i)
			continue
		}

		assert.EqualValues(t, &Prefix{
			Name: test.Name,
			User: test.User,
			Host: test.Host,
		}, pi, "%d. Identity did not match", i)
	}
}

func BenchmarkParsePrefix(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParsePrefix(messageTests[i%len(messageTests)].Expect)
	}
}

func TestMessageTrailing(t *testing.T) {
	t.Parallel()

	for i, test := range messageTests {
		if test.Err != "" {
			continue
		}

		m, _ := ParseMessage(test.Expect)
		tr := m.Trailing()
		if len(test.Params) < 1 {
			assert.Equal(t, "", tr, "%d. Expected empty trailing", i)
		} else {
			assert.Equal(t, test.Params[len(test.Params)-1], tr, "%d. Expected matching traling", i)
		}
	}
}

func TestMessageFromChan(t *testing.T) {
	t.Parallel()

	for i, test := range messageTests {
		if test.Err != "" {
			continue
		}

		m, _ := ParseMessage(test.Expect)
		assert.Equal(t, test.FromChan, m.FromChannel(), "%d. Wrong FromChannel value", i)
	}
}

func TestMessageCopy(t *testing.T) {
	t.Parallel()

	for i, test := range messageTests {
		if test.Err != "" {
			continue
		}

		m, _ := ParseMessage(test.Expect)

		c := m.Copy()
		assert.EqualValues(t, m, c, "%d. Copied values are not equal", i)

		if len(m.Tags) > 0 {
			c = m.Copy()
			for k := range c.Tags {
				c.Tags[k] += "junk"
			}

			assert.False(t, assert.ObjectsAreEqualValues(m, c), "%d. Copied with modified tags should not match", i)
		}

		c = m.Copy()
		c.Prefix.Name += "junk"
		assert.False(t, assert.ObjectsAreEqualValues(m, c), "%d. Copied with modified identity should not match", i)

		c = m.Copy()
		c.Params = append(c.Params, "junk")
		assert.False(t, assert.ObjectsAreEqualValues(m, c), "%d. Copied with additional params should not match", i)
	}

	// The message itself doesn't matter, we just need to make sure we
	// don't error if the user does something crazy and makes Params
	// nil.
	m, _ := ParseMessage("PING :hello world")
	m.Prefix = nil
	c := m.Copy()

	assert.EqualValues(t, m, c, "nil prefix copy failed")
}

func TestMessageString(t *testing.T) {
	t.Parallel()

	for i, test := range messageTests {
		if test.Err != "" {
			continue
		}

		m, _ := ParseMessage(test.Expect)
		if test.ExpectIn != nil {
			assert.Contains(t, test.ExpectIn, m.String()+"\n", "%d. Message Stringification failed", i)
		} else {
			assert.Equal(t, test.Expect, m.String()+"\n", "%d. Message Stringification failed", i)
		}
	}
}

func TestMessageTags(t *testing.T) {
	t.Parallel()

	for i, test := range messageTests {
		if test.Err != "" || test.Tags == nil {
			continue
		}

		m, _ := ParseMessage(test.Expect)
		assert.EqualValues(t, test.Tags, m.Tags, "%d. Tag parsing failed", i)

		// Ensure we have all the tags we expected.
		for k, v := range test.Tags {
			tag, ok := m.GetTag(k)
			assert.True(t, ok, "%d. Missing tag %q", i, k)
			assert.EqualValues(t, v, tag, "%d. Wrong tag value", i)
		}

		assert.EqualValues(t, test.Tags, m.Tags, "%d. Tags don't match", i)
	}
}

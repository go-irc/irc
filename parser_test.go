package irc

import (
	"io/ioutil"
	"strings"
	"testing"

	yaml "gopkg.in/yaml.v2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	Err      error

	// FromChannel
	FromChan bool
}{
	{ // Empty message should error
		Err: ErrZeroLengthMessage,
	},
	{ // Make sure we've got a command
		Expect: ":asd  :",
		Err:    ErrMissingCommand,
	},
	{ // Need data after tags
		Expect: "@A",
		Err:    ErrMissingDataAfterTags,
	},
	{ // Need data after prefix
		Expect: ":A",
		Err:    ErrMissingDataAfterPrefix,
	},
	{ // Basic prefix test
		Prefix: "server.kevlar.net",
		Cmd:    "PING",

		Name: "server.kevlar.net",

		Expect: ":server.kevlar.net PING\n",
	},
	{ // Trailing argument test
		Prefix: "server.kevlar.net",
		Cmd:    "NOTICE",
		Params: []string{"user", "*** This is a test"},

		Name: "server.kevlar.net",

		Expect: ":server.kevlar.net NOTICE user :*** This is a test\n",
	},
	{ // Full prefix test
		Prefix: "belakA!belakB@a.host.com",
		Cmd:    "PRIVMSG",
		Params: []string{"#somewhere", "*** This is a test"},

		Name: "belakA",
		User: "belakB",
		Host: "a.host.com",

		Expect:   ":belakA!belakB@a.host.com PRIVMSG #somewhere :*** This is a test\n",
		FromChan: true,
	},
	{ // Test : in the middle of a param
		Prefix: "freenode",
		Cmd:    "005",
		Params: []string{"starkbot", "CHANLIMIT=#:120", "MORE", "are supported by this server"},

		Name: "freenode",

		Expect: ":freenode 005 starkbot CHANLIMIT=#:120 MORE :are supported by this server\n",
	},
	{ // Test FromChannel on a different channel prefix
		Prefix: "belakA!belakB@a.host.com",
		Cmd:    "PRIVMSG",
		Params: []string{"&somewhere", "*** This is a test"},

		Name: "belakA",
		User: "belakB",
		Host: "a.host.com",

		Expect:   ":belakA!belakB@a.host.com PRIVMSG &somewhere :*** This is a test\n",
		FromChan: true,
	},
	{ // Test FromChannel on a single user
		Prefix: "belakA!belakB@a.host.com",
		Cmd:    "PRIVMSG",
		Params: []string{"belak", "*** This is a test"},

		Name: "belakA",
		User: "belakB",
		Host: "a.host.com",

		Expect: ":belakA!belakB@a.host.com PRIVMSG belak :*** This is a test\n",
	},
	{ // Simple message
		Cmd:    "B",
		Params: []string{"C"},
		Expect: "B C\n",
	},
	{ // Simple message with tags
		Prefix: "A@B",
		Cmd:    "C",
		Params: []string{"D"},

		Name: "A",
		Host: "B",

		Expect: ":A@B C D\n",
	},
	{ // Simple message with prefix
		Prefix: "A",
		Cmd:    "B",
		Params: []string{"C"},

		Name: "A",

		Expect: ":A B C\n",
	},
	{ // Message with prefix and multiple params
		Prefix: "A",
		Cmd:    "B",
		Params: []string{"C", "D"},

		Name: "A",

		Expect: ":A B C D\n",
	},
	{ // Message with empty trailing
		Cmd:    "A",
		Params: []string{""},

		Expect: "A :\n",
	},
	{ // Test basic tag parsing
		Tags: Tags{
			"tag": "value",
		},

		Cmd: "A",

		Expect: "@tag=value A\n",
	},
	{ // Escaped \n in tag
		Tags: Tags{
			"tag": "\n",
		},

		Cmd: "A",

		Expect: "@tag=\\n A\n",
	},
	{ // Escaped \ in tag
		Tags: Tags{
			"tag": "\\",
		},

		Cmd: "A",

		Expect:   "@tag=\\ A\n",
		ExpectIn: []string{"@tag=\\\\ A\n"},
	},
	{ // Escaped ; in tag
		Tags: Tags{
			"tag": ";",
		},

		Cmd: "A",

		Expect: "@tag=\\: A\n",
	},
	{ // Empty tag
		Tags: Tags{
			"tag": "",
		},

		Cmd: "A",

		Expect: "@tag A\n",
	},
	{ // Escaped & in tag
		Tags: Tags{
			"tag": "\\&",
		},

		Cmd: "A",

		Expect:   "@tag=\\& A\n",
		ExpectIn: []string{"@tag=\\\\& A\n"},
	},
	{ // Multiple simple tags
		Tags: Tags{
			"tag":  "x",
			"tag2": "asd",
		},

		Cmd: "A",

		Expect:   "@tag=x;tag2=asd A\n",
		ExpectIn: []string{"@tag=x;tag2=asd A\n", "@tag2=asd;tag=x A\n"},
	},
	{ // Complicated escaped tag
		Tags: Tags{
			"tag": "; \\\r\n",
		},

		Cmd:    "A",
		Expect: "@tag=\\:\\s\\\\\\r\\n A\n",
	},
	{ // Tags example from the spec
		Tags: Tags{
			"aaa":             "bbb",
			"ccc":             "",
			"example.com/ddd": "eee",
		},

		Name: "nick",
		User: "ident",
		Host: "host.com",

		Prefix: "nick!ident@host.com",
		Cmd:    "PRIVMSG",
		Params: []string{"me", "Hello"},

		Expect: "@aaa=bbb;ccc;example.com/ddd=eee :nick!ident@host.com PRIVMSG me :Hello\n",
		ExpectIn: []string{
			"@aaa=bbb;ccc;example.com/ddd=eee :nick!ident@host.com PRIVMSG me Hello\n",
			"@aaa=bbb;example.com/ddd=eee;ccc :nick!ident@host.com PRIVMSG me Hello\n",
			"@ccc;aaa=bbb;example.com/ddd=eee :nick!ident@host.com PRIVMSG me Hello\n",
			"@ccc;example.com/ddd=eee;aaa=bbb :nick!ident@host.com PRIVMSG me Hello\n",
			"@example.com/ddd=eee;aaa=bbb;ccc :nick!ident@host.com PRIVMSG me Hello\n",
			"@example.com/ddd=eee;ccc;aaa=bbb :nick!ident@host.com PRIVMSG me Hello\n",
		},
	},
	{ // = in tag
		Tags: Tags{
			"a": "a=a",
		},

		Name: "nick",
		User: "ident",
		Host: "host.com",

		Prefix: "nick!ident@host.com",
		Cmd:    "PRIVMSG",
		Params: []string{"me", "Hello"},

		Expect:   "@a=a=a :nick!ident@host.com PRIVMSG me :Hello\n",
		ExpectIn: []string{"@a=a=a :nick!ident@host.com PRIVMSG me Hello\n"},
	},
}

func TestMustParseMessage(t *testing.T) {
	t.Parallel()

	for i, test := range messageTests {
		if test.Err != nil {
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
		if test.Err != nil {
			assert.Equal(t, test.Err, err, "%d. Didn't get correct error for invalid message.", i)
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
		if test.Err != nil {
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
		if test.Err != nil {
			continue
		}

		m, _ := ParseMessage(test.Expect)
		assert.Equal(t, test.FromChan, m.FromChannel(), "%d. Wrong FromChannel value", i)
	}
}

func TestMessageCopy(t *testing.T) {
	t.Parallel()

	for i, test := range messageTests {
		if test.Err != nil {
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
		if test.Err != nil {
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
		if test.Err != nil || test.Tags == nil {
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

type MsgSplitTests struct {
	Tests []struct {
		Input string
		Atoms struct {
			Source *string
			Verb   string
			Params []string
			Tags   map[string]interface{}
		}
	}
}

func TestMsgSplit(t *testing.T) {
	data, err := ioutil.ReadFile("./testcases/tests/msg-split.yaml")
	require.NoError(t, err)

	var splitTests MsgSplitTests
	err = yaml.Unmarshal(data, &splitTests)
	require.NoError(t, err)

	for _, test := range splitTests.Tests {
		msg, err := ParseMessage(test.Input)
		assert.NoError(t, err, "Failed to parse: %s (%s)", test.Input, err)

		assert.Equal(t,
			strings.ToUpper(test.Atoms.Verb), msg.Command,
			"Wrong command for input: %s", test.Input,
		)
		assert.Equal(t,
			test.Atoms.Params, msg.Params,
			"Wrong params for input: %s", test.Input,
		)

		if test.Atoms.Source != nil {
			assert.Equal(t, *test.Atoms.Source, msg.Prefix.String())
		}

		assert.Equal(t,
			len(test.Atoms.Tags), len(msg.Tags),
			"Wrong number of tags",
		)

		for k, v := range test.Atoms.Tags {
			if v == nil {
				assert.EqualValues(t, "", msg.Tags[k], "Tag differs")
			} else {
				assert.EqualValues(t, v, msg.Tags[k], "Tag differs")
			}
		}
	}
}

type MsgJoinTests struct {
	Tests []struct {
		Atoms struct {
			Source string
			Verb   string
			Params []string
			Tags   map[string]interface{}
		}
		Matches []string
	}
}

func TestMsgJoin(t *testing.T) {
	data, err := ioutil.ReadFile("./testcases/tests/msg-join.yaml")
	require.NoError(t, err)

	var splitTests MsgJoinTests
	err = yaml.Unmarshal(data, &splitTests)
	require.NoError(t, err)

	for _, test := range splitTests.Tests {
		msg := &Message{
			Prefix:  ParsePrefix(test.Atoms.Source),
			Command: test.Atoms.Verb,
			Params:  test.Atoms.Params,
			Tags:    make(map[string]TagValue),
		}

		for k, v := range test.Atoms.Tags {
			if v == nil {
				msg.Tags[k] = TagValue("")
			} else {
				msg.Tags[k] = TagValue(v.(string))
			}
		}

		assert.Contains(t, test.Matches, msg.String())
	}
}

type UserhostSplitTests struct {
	Tests []struct {
		Source string
		Atoms  struct {
			Nick string
			User string
			Host string
		}
	}
}

func TestUserhostSplit(t *testing.T) {
	data, err := ioutil.ReadFile("./testcases/tests/userhost-split.yaml")
	require.NoError(t, err)

	var userhostTests UserhostSplitTests
	err = yaml.Unmarshal(data, &userhostTests)
	require.NoError(t, err)

	for _, test := range userhostTests.Tests {
		prefix := ParsePrefix(test.Source)

		assert.Equal(t,
			test.Atoms.Nick, prefix.Name,
			"Name did not match for input: %q", test.Source,
		)
		assert.Equal(t,
			test.Atoms.User, prefix.User,
			"User did not match for input: %q", test.Source,
		)
		assert.Equal(t,
			test.Atoms.Host, prefix.Host,
			"Host did not match for input: %q", test.Source,
		)
	}
}

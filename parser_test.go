package irc

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func BenchmarkParseMessage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		MustParseMessage("@tag1=something :nick!user@host PRIVMSG #channel :some message")
	}
}

func TestParseMessage(t *testing.T) {
	t.Parallel()

	var messageTests = []struct {
		Input string
		Err   error
	}{
		{
			Input: "",
			Err:   ErrZeroLengthMessage,
		},
		{
			Input: "@asdf",
			Err:   ErrMissingDataAfterTags,
		},
		{
			Input: ":asdf",
			Err:   ErrMissingDataAfterPrefix,
		},
		{
			Input: " :",
			Err:   ErrMissingCommand,
		},
		{
			Input: "PING :asdf",
		},
	}

	for i, test := range messageTests {
		m, err := ParseMessage(test.Input)
		assert.Equal(t, test.Err, err, "%d. Error didn't match expected", i)

		if test.Err != nil {
			assert.Nil(t, m, "%d. Didn't get nil message", i)
		} else {
			assert.NotNil(t, m, "%d. Got nil message", i)
		}
	}
}

func TestMustParseMessage(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		MustParseMessage("")
	}, "Didn't get expected panic")

	assert.NotPanics(t, func() {
		MustParseMessage("PING :asdf")
	}, "Got unexpected panic")
}

func TestMessageParam(t *testing.T) {
	t.Parallel()

	m := MustParseMessage("PING :test")
	assert.Equal(t, m.Param(0), "test")
	assert.Equal(t, m.Param(-1), "")
	assert.Equal(t, m.Param(2), "")
}

func TestMessageTrailing(t *testing.T) {
	t.Parallel()

	m := MustParseMessage("PING :helloworld")
	assert.Equal(t, "helloworld", m.Trailing())

	m = MustParseMessage("PING")
	assert.Equal(t, "", m.Trailing())
}

func TestMessageCopy(t *testing.T) {
	t.Parallel()

	m := MustParseMessage("@tag=val :user@host PING :helloworld")

	// Ensure copied messages are equal
	c := m.Copy()
	assert.EqualValues(t, m, c, "Copied values are not equal")

	// Ensure messages with modified tags don't match
	c = m.Copy()
	for k := range c.Tags {
		c.Tags[k] += "junk"
	}
	assert.False(t, assert.ObjectsAreEqualValues(m, c), "Copied with modified tags should not match")

	// Ensure messages with modified prefix don't match
	c = m.Copy()
	c.Prefix.Name += "junk"
	assert.False(t, assert.ObjectsAreEqualValues(m, c), "Copied with modified identity should not match")

	// Ensure messages with modified params don't match
	c = m.Copy()
	c.Params = append(c.Params, "junk")
	assert.False(t, assert.ObjectsAreEqualValues(m, c), "Copied with additional params should not match")

	// The message itself doesn't matter, we just need to make sure we
	// don't error if the user does something crazy and makes Params
	// nil.
	m = MustParseMessage("PING :hello world")
	m.Prefix = nil
	c = m.Copy()
	assert.EqualValues(t, m, c, "nil prefix copy failed")

	// Ensure an empty Params is copied as nil
	m = MustParseMessage("PING")
	m.Params = []string{}
	c = m.Copy()
	assert.Nil(t, c.Params, "Expected nil for empty params")
}

// Everything beyond here comes from the testcases repo

type MsgSplitTests struct {
	Tests []struct {
		Desc  string
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
	t.Parallel()

	data, err := ioutil.ReadFile("./_testcases/tests/msg-split.yaml")
	require.NoError(t, err)

	var splitTests MsgSplitTests
	err = yaml.Unmarshal(data, &splitTests)
	require.NoError(t, err)

	for _, test := range splitTests.Tests {
		msg, err := ParseMessage(test.Input)
		assert.NoError(t, err, "%s: Failed to parse: %s (%s)", test.Desc, test.Input, err)

		assert.Equal(t,
			strings.ToUpper(test.Atoms.Verb), msg.Command,
			"%s: Wrong command for input: %s", test.Desc, test.Input,
		)
		assert.Equal(t,
			test.Atoms.Params, msg.Params,
			"%s: Wrong params for input: %s", test.Desc, test.Input,
		)

		if test.Atoms.Source != nil {
			assert.Equal(t, *test.Atoms.Source, msg.Prefix.String())
		}

		assert.Equal(t,
			len(test.Atoms.Tags), len(msg.Tags),
			"%s: Wrong number of tags",
			test.Desc,
		)

		for k, v := range test.Atoms.Tags {
			tag, ok := msg.GetTag(k)
			assert.True(t, ok, "Missing tag")
			if v == nil {
				assert.EqualValues(t, "", tag, "%s: Tag %q differs: %s != \"\"", test.Desc, k, tag)
			} else {
				assert.EqualValues(t, v, tag, "%s: Tag %q differs: %s != %s", test.Desc, k, v, tag)
			}
		}
	}
}

type MsgJoinTests struct {
	Tests []struct {
		Desc  string
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
	t.Parallel()

	data, err := ioutil.ReadFile("./_testcases/tests/msg-join.yaml")
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
		Desc   string
		Source string
		Atoms  struct {
			Nick string
			User string
			Host string
		}
	}
}

func TestUserhostSplit(t *testing.T) {
	t.Parallel()

	data, err := ioutil.ReadFile("./_testcases/tests/userhost-split.yaml")
	require.NoError(t, err)

	var userhostTests UserhostSplitTests
	err = yaml.Unmarshal(data, &userhostTests)
	require.NoError(t, err)

	for _, test := range userhostTests.Tests {
		prefix := ParsePrefix(test.Source)

		assert.Equal(t,
			test.Atoms.Nick, prefix.Name,
			"%s: Name did not match for input: %q", test.Desc, test.Source,
		)
		assert.Equal(t,
			test.Atoms.User, prefix.User,
			"%s: User did not match for input: %q", test.Desc, test.Source,
		)
		assert.Equal(t,
			test.Atoms.Host, prefix.Host,
			"%s: Host did not match for input: %q", test.Desc, test.Source,
		)
	}
}

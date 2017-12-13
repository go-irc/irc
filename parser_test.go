package irc

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestMustParseMessage(t *testing.T) {
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
			assert.Panics(t, func() {
				MustParseMessage(test.Input)
			}, "%d. Didn't get expected panic", i)

			assert.Nil(t, m, "%d. Didn't get nil message", i)
		} else {
			assert.NotPanics(t, func() {
				MustParseMessage(test.Input)
			}, "%d. Got unexpected panic", i)

			assert.NotNil(t, m, "%d. Got nil message", i)
		}
	}
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
			tag, ok := msg.GetTag(k)
			assert.True(t, ok, "Missing tag")
			if v == nil {
				assert.EqualValues(t, "", tag, "Tag differs")
			} else {
				assert.EqualValues(t, v, tag, "Tag differs")
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
	t.Parallel()

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
	t.Parallel()

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

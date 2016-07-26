package irc

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

var messageTests = []struct {
	// Message parsing
	Prefix, Cmd string
	Params      []string

	// Prefix parsing
	Name, User, Host string

	// Total output
	Expect string
	IsNil  bool

	// FromChannel
	FromChan bool
}{
	{
		IsNil: true,
	},
	{
		Expect: ":asd  :",
		IsNil:  true,
	},
	{
		Expect: ":A",
		IsNil:  true,
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
}

func TestParseMessage(t *testing.T) {
	t.Parallel()

	for i, test := range messageTests {
		m := ParseMessage(test.Expect)
		if test.IsNil {
			assert.Nil(t, m, "%d. Didn't get nil for invalid message.", i)
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
		// TODO: Not sure if we should be skipping empty strings or handling them.
		if test.Prefix == "" {
			continue
		}

		pi := ParsePrefix(test.Prefix)
		if pi == nil {
			t.Errorf("%d. Got nil for valid identity", i)
			continue
		}
		if test.Name != pi.Name {
			t.Errorf("%d. name = %q, want %q", i, pi.Name, test.Name)
		}
		if test.User != pi.User {
			t.Errorf("%d. user = %q, want %q", i, pi.User, test.User)
		}
		if test.Host != pi.Host {
			t.Errorf("%d. host = %q, want %q", i, pi.Host, test.Host)
		}
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
		if test.IsNil {
			continue
		}

		m := ParseMessage(test.Expect)
		tr := m.Trailing()
		if len(test.Params) < 1 {
			if tr != "" {
				t.Errorf("%d. trailing = %q, want %q", i, tr, "")
			}
		} else if tr != test.Params[len(test.Params)-1] {
			t.Errorf("%d. trailing = %q, want %q", i, tr, test.Params[len(test.Params)-1])
		}
	}
}

func TestMessageFromChan(t *testing.T) {
	t.Parallel()

	for i, test := range messageTests {
		if test.IsNil {
			continue
		}

		m := ParseMessage(test.Expect)
		if m.FromChannel() != test.FromChan {
			t.Errorf("%d. fromchannel = %v, want %v", i, m.FromChannel(), test.FromChan)
		}
	}
}

func TestMessageCopy(t *testing.T) {
	t.Parallel()

	for i, test := range messageTests {
		if test.IsNil {
			continue
		}

		m := ParseMessage(test.Expect)
		c := m.Copy()

		if !reflect.DeepEqual(m, c) {
			t.Errorf("%d. copy = %q, want %q", i, m, c)
		}

		if c.Prefix != nil {
			c.Prefix.Name += "junk"
			if reflect.DeepEqual(m, c) {
				t.Errorf("%d. copyidentity matched when it shouldn't", i)
			}
		}

		c.Params = append(c.Params, "junk")
		if reflect.DeepEqual(m, c) {
			t.Errorf("%d. copyargs matched when it shouldn't", i)
		}
	}
}

func TestMessageString(t *testing.T) {
	t.Parallel()

	for i, test := range messageTests {
		if test.IsNil {
			continue
		}

		m := ParseMessage(test.Expect)
		if m.String()+"\n" != test.Expect {
			t.Errorf("%d. %s did not match %s", i, m.String(), test.Expect)
		}
	}
}

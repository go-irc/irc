package irc

import (
	"reflect"
	"testing"
)

var eventTests = []struct {
	// Event parsing
	Prefix, Cmd string
	Args        []string

	// Identity parsing
	Nick, User, Host string

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
		Expect: ":A",
		IsNil:  true,
	},
	{
		Prefix: "server.kevlar.net",
		Cmd:    "PING",

		Host: "server.kevlar.net",

		Expect: ":server.kevlar.net PING\n",
	},
	{
		Prefix: "server.kevlar.net",
		Cmd:    "NOTICE",
		Args:   []string{"user", "*** This is a test"},

		Host: "server.kevlar.net",

		Expect: ":server.kevlar.net NOTICE user :*** This is a test\n",
	},
	{
		Prefix: "belakA!belakB@a.host.com",
		Cmd:    "PRIVMSG",
		Args:   []string{"#somewhere", "*** This is a test"},

		Nick: "belakA",
		User: "belakB",
		Host: "a.host.com",

		Expect:   ":belakA!belakB@a.host.com PRIVMSG #somewhere :*** This is a test\n",
		FromChan: true,
	},
	{
		Prefix: "belakA!belakB@a.host.com",
		Cmd:    "PRIVMSG",
		Args:   []string{"&somewhere", "*** This is a test"},

		Nick: "belakA",
		User: "belakB",
		Host: "a.host.com",

		Expect:   ":belakA!belakB@a.host.com PRIVMSG &somewhere :*** This is a test\n",
		FromChan: true,
	},
	{
		Prefix: "belakA!belakB@a.host.com",
		Cmd:    "PRIVMSG",
		Args:   []string{"belak", "*** This is a test"},

		Nick: "belakA",
		User: "belakB",
		Host: "a.host.com",

		Expect: ":belakA!belakB@a.host.com PRIVMSG belak :*** This is a test\n",
	},
	{
		Prefix: "A",
		Cmd:    "B",
		Args:   []string{"C"},

		Host: "A",

		Expect: ":A B C\n",
	},
	{
		Prefix: "A@B",
		Cmd:    "C",
		Args:   []string{"D"},

		User: "A",
		Host: "B",

		Expect: ":A@B C D\n",
	},
	{
		Cmd:    "B",
		Args:   []string{"C"},
		Expect: "B C\n",
	},
	{
		Prefix: "A",
		Cmd:    "B",
		Args:   []string{"C", "D"},

		Host: "A",

		Expect: ":A B C D\n",
	},
}

func TestParseEvent(t *testing.T) {
	for i, test := range eventTests {
		e := ParseEvent(test.Expect)
		if e == nil && !test.IsNil {
			t.Errorf("%d. Got nil for valid event", i)
		} else if e != nil && test.IsNil {
			t.Errorf("%d. Didn't get nil for invalid event", i)
		}

		if e == nil {
			continue
		}

		if test.Prefix != e.Prefix {
			t.Errorf("%d. prefix = %q, want %q", i, e.Prefix, test.Prefix)
		}
		if test.Cmd != e.Command {
			t.Errorf("%d. command = %q, want %q", i, e.Command, test.Cmd)
		}
		if len(test.Args) != len(e.Args) {
			t.Errorf("%d. args = %v, want %v", i, e.Args, test.Args)
		} else {
			for j := 0; j < len(test.Args) && j < len(e.Args); j++ {
				if test.Args[j] != e.Args[j] {
					t.Errorf("%d. arg[%d] = %q, want %q", i, e.Args[j], test.Args[j])
				}
			}
		}
	}
}

func BenchmarkParseEvent(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseEvent(eventTests[i%len(eventTests)].Prefix)
	}
}

func TestParseIdentity(t *testing.T) {
	for i, test := range eventTests {
		// TODO: Not sure if we should be skipping empty strings or handling them.
		if test.Prefix == "" {
			continue
		}

		pi := ParseIdentity(test.Prefix)
		if pi == nil {
			t.Errorf("%d. Got nil for valid identity", pi)
			continue
		}
		if test.Nick != pi.Nick {
			t.Errorf("%d. nick = %q, want %q", i, pi.Nick, test.Nick)
		}
		if test.User != pi.User {
			t.Errorf("%d. user = %q, want %q", i, pi.User, test.User)
		}
		if test.Host != pi.Host {
			t.Errorf("%d. host = %q, want %q", i, pi.Host, test.Host)
		}
	}
}

func BenchmarkParseIdentity(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseIdentity(eventTests[i%len(eventTests)].Expect)
	}
}

func TestEventTrailing(t *testing.T) {
	for i, test := range eventTests {
		if test.IsNil {
			continue
		}

		e := ParseEvent(test.Expect)
		tr := e.Trailing()
		if len(test.Args) < 1 {
			if tr != "" {
				t.Errorf("%d. trailing = %q, want %q", i, tr, "")
			}
		} else if tr != test.Args[len(test.Args)-1] {
			t.Errorf("%d. trailing = %q, want %q", i, tr, test.Args[len(test.Args)-1])
		}
	}
}

func TestEventFromChan(t *testing.T) {
	for i, test := range eventTests {
		if test.IsNil {
			continue
		}

		e := ParseEvent(test.Expect)
		if e.FromChannel() != test.FromChan {
			t.Errorf("%d. fromchannel = %q, want %q", i, e.FromChannel(), test.FromChan)
		}
	}
}

func TestEventCopy(t *testing.T) {
	for i, test := range eventTests {
		if test.IsNil {
			continue
		}

		e := ParseEvent(test.Expect)
		c := e.Copy()

		if !reflect.DeepEqual(e, c) {
			t.Errorf("%d. copy = %q, want %q", i, e, c)
		}

		if c.Identity != nil {
			c.Identity.Nick += "junk"
			if reflect.DeepEqual(e, c) {
				t.Errorf("%d. copyidentity matched when it shouldn't", i)
			}
		}

		c.Args = append(c.Args, "junk")
		if reflect.DeepEqual(e, c) {
			t.Errorf("%d. copyargs matched when it shouldn't", i)
		}
	}
}

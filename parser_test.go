package irc

import "testing"

var eventTests = []struct {
	Prefix, Cmd string
	Args        []string
	Expect      string
}{
	{
		Prefix: "server.kevlar.net",
		Cmd:    "NOTICE",
		Args:   []string{"user", "*** This is a test"},
		Expect: ":server.kevlar.net NOTICE user :*** This is a test\n",
	},
	{
		Prefix: "A",
		Cmd:    "B",
		Args:   []string{"C"},
		Expect: ":A B C\n",
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
		Expect: ":A B C D\n",
	},
}

func TestParseEvent(t *testing.T) {
	for i, test := range eventTests {
		e := ParseEvent(test.Expect)
		if e == nil {
			t.Errorf("%d. Got nil for valid event", i)
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
		ParseEvent(eventTests[i%len(eventTests)].Expect)
	}
}

var identityTests = []struct {
	Nick, User, Host string
	Expect           string
}{
	{
		Nick:   "NickServA",
		User:   "NickServB",
		Host:   "services",
		Expect: "NickServA!NickServB@services",
	},
	{
		User:   "NickServ",
		Host:   "services",
		Expect: "NickServ@services",
	},
	{
		Host:   "NickServ",
		Expect: "NickServ",
	},
}

func TestParseIdentity(t *testing.T) {
	for i, test := range identityTests {
		pi := ParseIdentity(test.Expect)
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
		ParseIdentity(identityTests[i%len(identityTests)].Expect)
	}
}

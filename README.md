# irc

[![Build Status](https://img.shields.io/travis/go-irc/irc.svg)](https://travis-ci.org/go-irc/irc)
[![Coverage Status](https://img.shields.io/coveralls/go-irc/irc.svg)](https://coveralls.io/github/go-irc/irc?branch=master)

irc is a simple, low-ish level golang irc library which is meant to
only read and write messages from a given stream. There are a number
of other libraries which provide a more full featured client if that's
what you're looking for. This library is more of a building block for
other things to build on.

While this library aims for forwards API compatibility, there may be
some breaking changes if something important comes up.

## Example

```go
package main

import (
	"log"
	"net"

	"github.com/belak/irc"
)

func main() {
	conn, err := net.Dial("tcp", "chat.freenode.net:6667")
	if err != nil {
		log.Fatalln(err)
	}

	config := irc.ClientConfig{
		Nick: "i_have_a_nick",
		Pass: "password",
		User: "username",
		Name: "Full Name",
		Handler: irc.HandlerFunc(func(c *irc.Client, m *irc.Message) {
			if m.Command == "001" {
				// 001 is a welcome event, so we join channels there
				c.Write("JOIN #bot-test-chan")
			} else if m.Command == "PRIVMSG" && m.FromChannel() {
				// Create a handler on all messages.
				c.WriteMessage(&irc.Message{
					Command: "PRIVMSG",
					Params: []string{
						m.Params[0],
						m.Trailing(),
					},
				})
			}
		}),
	}

	// Create the client
	client := irc.NewClient(conn, config)
	err = client.Run()
	if err != nil {
		log.Fatalln(err)
	}
}
```

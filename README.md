# go-irc

[![GoDoc](https://img.shields.io/badge/doc-GoDoc-blue.svg)](https://godoc.org/github.com/go-irc/irc)
[![Build Status](https://img.shields.io/travis/go-irc/irc.svg)](https://travis-ci.org/go-irc/irc)
[![Coverage Status](https://img.shields.io/coveralls/go-irc/irc.svg)](https://coveralls.io/github/go-irc/irc?branch=master)

This package was originally created to only handle message parsing,
but has since been expanded to include a small abstraction around a
connection and a simple client.

This library is not designed to hide any of the IRC elements from
you. If you just want to build a simple chat bot and don't want to
deal with IRC in particular, there are a number of other libraries
which provide a more full featured client if that's what you're
looking for.

This library is meant to stay as simple as possible so it can be a
building block for other packages.

This library aims for API compatibility whenever possible. New
functions and other additions will most likely not result in a major
version increase unless they break the API. This library aims to
follow the semver recommendations mentioned on gopkg.in.

## Import and Branches

Currently there are two main branches. `master` is where most
development happens and will include fixes and new features the
quickest. `v1` can be considered the stable branch. `master` will be
frequently merged into `v1` when it is considered to be stable.

As a result of the two branches, there are multiple import locations.

* `gopkg.in/irc.v1` should be used to develop against the stable branch
* `github.com/go-irc/irc` should be used to develop against the master branch

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

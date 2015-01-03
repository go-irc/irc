# irc

[![Build Status](https://travis-ci.org/belak/irc.svg?branch=master)](https://travis-ci.org/belak/irc)

irc is a simple golang irc library based around the idea of simple handlers.

## Usage

```
package main

import (
	"github.com/belak/irc"
	"log"
)

func main() {
	// Create a mux to handle different events
	handler := irc.NewBasicMux()

	// Create the client
	client := irc.NewClient(irc.HandlerFunc(handler.HandleEvent), "i_have_a_nick", "user", "name", "pass")

	// 001 is a welcome event, so we join channels there
	handler.Event("001", func(c *irc.Client, e *irc.Event) {
		c.Write("JOIN #bot-test-chan")
	})

	// Create a handler on all messages.
	handler.Event("PRIVMSG", func(c *irc.Client, e *irc.Event) {
		c.MentionReply(e, e.Trailing())
	})

	// Connect to the server
	// Note that the framework does not currently handle reconnecting
	err := client.Dial("chat.freenode.net:6667")
	if err != nil {
		log.Fatalln(err)
	}
}
```

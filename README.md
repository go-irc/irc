# irc

[![Build Status](https://travis-ci.org/belak/irc.svg?branch=master)](https://travis-ci.org/belak/irc)
[![Coverage Status](https://coveralls.io/repos/belak/irc/badge.svg?branch=master&service=github)](https://coveralls.io/github/belak/irc?branch=master)

irc is a simple, low-ish level golang irc library which is meant to
only read and write messages from a given stream. There are a number
of other libraries which provide a more full featured client if that's
what you're looking for. This library is more of a building block for
other things to build on.

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

        // Create the client
        client := irc.NewClient(conn, "i_have_a_nick", "user", "name", "pass")

        for {
                m, err := client.ReadMessage()
                if err != nil {
                        log.Fatalln(err)
                }

                if m.Command == "001" {
                        // 001 is a welcome event, so we join channels there
                        c.Write("JOIN #bot-test-chan")
                } else if m.Command == "PRIVMSG" {
                        // Create a handler on all messages.
                        c.MentionReply(e, e.Trailing())
                }
        }
}
```

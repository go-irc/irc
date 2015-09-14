package irc

import (
	"strings"
	"sync"
	"unicode"
)

// MentionMux is a simple IRC event multiplexer, based on a slice of Handlers
//
// The MentionMux uses the current Nick and punctuation to determine if the
// Client has been mentioned. The nick, punctuation and any leading or
// trailing spaces are removed from the message.
type MentionMux struct {
	handlers []HandlerFunc
	lock     *sync.RWMutex
}

// NewMentionMux will create an initialized MentionMux with no handlers.
func NewMentionMux() *MentionMux {
	return &MentionMux{
		nil,
		&sync.RWMutex{},
	}
}

// Event will register a Handler
func (m *MentionMux) Event(h HandlerFunc) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.handlers = append(m.handlers, h)
}

// HandleEvent strips off the nick punctuation and spaces and runs the handlers
func (m *MentionMux) HandleEvent(c *Client, e *Event) {
	if e.Command != "PRIVMSG" {
		// TODO: Log this
		return
	}

	lastArg := e.Trailing()
	nick := c.currentNick

	// We only handle this event if it starts with the
	// current bot's nick followed by punctuation
	if len(lastArg) < len(nick)+2 ||
		!strings.HasPrefix(lastArg, nick) ||
		!unicode.IsPunct(rune(lastArg[len(nick)])) ||
		lastArg[len(nick)+1] != ' ' {

		return
	}

	// Copy it into a new Event
	newEvent := e.Copy()

	// Strip the nick, punctuation, and spaces from the message
	newEvent.Args[len(newEvent.Args)-1] = strings.TrimSpace(lastArg[len(nick)+1:])

	m.lock.RLock()
	defer m.lock.RUnlock()

	for _, h := range m.handlers {
		h(c, newEvent)
	}
}

package bot

import (
	"sync"

	"github.com/belak/irc"
)

// BasicMux is a simple IRC event multiplexer.
// It matches the command against registered Handlers and calls the correct set.
//
// Handlers will be processed in the order in which they were added.
// Registering a handler with a "*" command will cause it to receive all events.
// Note that even though "*" will match all commands, glob matching is not used.
type BasicMux struct {
	m  map[string][]BotFunc
	mu *sync.Mutex
}

// This will create an initialized BasicMux with no handlers.
func NewBasicMux() *BasicMux {
	return &BasicMux{
		make(map[string][]BotFunc),
		&sync.Mutex{},
	}
}

// BasicMux.Event will register a Handler
func (m *BasicMux) Event(c string, h BotFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.m[c] = append(m.m[c], h)
}

// HandleEvent allows us to be a Handler so we can nest BasicMuxes
//
// The BasicMux simply dispatches all the Handler commands as needed
func (m *BasicMux) HandleEvent(b *Bot, e *irc.Event) {
	// Lock our handlers so we don't crap bricks if a
	// handler is added or removed from under our feet.
	m.mu.Lock()
	defer m.mu.Unlock()

	// Star means ALL THE THINGS
	// Really, this is only useful for logging
	for _, h := range m.m["*"] {
		h(b, e)
	}

	// Now that we've done the global handlers, we can run
	// the ones specific to this command
	for _, h := range m.m[e.Command] {
		h(b, e)
	}
}

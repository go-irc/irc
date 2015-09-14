package irc

import (
	"sort"
	"strings"
)

// CommandMux is a simple IRC event multiplexer, built on top of the BasicMux.

// HelpInfo is a collection of instructions for command usage that
// is formatted with <prefix>help
type HelpInfo struct {
	Usage       string
	Description string
}

// The CommandMux is given a prefix string and matches all PRIVMSG
// events which start with it. The first word after the string is
// moved into the Event.Command.
type CommandMux struct {
	private *BasicMux
	public  *BasicMux
	prefix  string
	cmdHelp map[string]*HelpInfo
}

// NewCommandMux will create an initialized BasicMux with no handlers.
func NewCommandMux(prefix string) *CommandMux {
	m := &CommandMux{
		NewBasicMux(),
		NewBasicMux(),
		prefix,
		make(map[string]*HelpInfo),
	}

	m.Event("help", m.help, &HelpInfo{
		"<command>",
		"Displays help messages for a given command",
	})
	return m
}

func (m *CommandMux) help(c *Client, e *Event) {
	cmd := e.Trailing()
	if cmd == "" {
		// Get all keys
		keys := make([]string, 0, len(m.cmdHelp))
		for k := range m.cmdHelp {
			keys = append(keys, k)
		}

		// Sort everything
		sort.Strings(keys)

		if e.FromChannel() {
			// If they said "!help" in a channel, list all available commands
			c.Reply(e, "Available commands: %s. Use %shelp [command] for more info.", strings.Join(keys, ", "), m.prefix)
		} else {
			for _, v := range keys {
				c.Reply(e, "%s: %s", v, m.cmdHelp[v])
			}
		}
	} else if help, ok := m.cmdHelp[cmd]; ok {
		if help == nil {
			c.Reply(e, "There is no help available for command %q", cmd)
		} else {
			lines := help.Format(m.prefix, cmd)
			for _, line := range lines {
				c.Reply(e, "%s", line)
			}
		}
	} else {
		c.MentionReply(e, "There is no help available for command %q", cmd)
	}
}

func (h *HelpInfo) Format(prefix, command string) []string {
	if h.Usage == "" && h.Description == "" {
		return []string{"There is no help available for command " + command}
	}

	ret := []string{}

	if h.Usage != "" {
		ret = append(ret, "Usage: "+prefix+command+" "+h.Usage)
	}

	if h.Description != "" {
		ret = append(ret, h.Description)
	}

	return ret
}

// Event will register a command handler for use as both a public and
// private command.
func (m *CommandMux) Event(c string, h HandlerFunc, help *HelpInfo) {
	m.private.Event(c, h)
	m.public.Event(c, h)

	m.cmdHelp[c] = help
}

// Channel will register a command handler for use as a public
// command.
func (m *CommandMux) Channel(c string, h HandlerFunc, help *HelpInfo) {
	m.public.Event(c, h)

	m.cmdHelp[c] = help
}

// Private will register a command handler for use as a private
// command.
func (m *CommandMux) Private(c string, h HandlerFunc, help *HelpInfo) {
	m.private.Event(c, h)

	m.cmdHelp[c] = help
}

// HandleEvent strips off the prefix, pulls the command out
// and runs HandleEvent on the internal BasicMux
func (m *CommandMux) HandleEvent(c *Client, e *Event) {
	if e.Command != "PRIVMSG" {
		// TODO: Log this
		return
	}

	// Get the last arg and see if it starts with the command prefix
	lastArg := e.Trailing()
	if !strings.HasPrefix(lastArg, m.prefix) {
		return
	}

	// Copy it into a new Event
	newEvent := &Event{}
	*newEvent = *e

	// Chop off the command itself
	msgParts := strings.SplitN(lastArg, " ", 2)
	newEvent.Args[len(newEvent.Args)-1] = ""
	if len(msgParts) > 1 {
		newEvent.Args[len(newEvent.Args)-1] = strings.TrimSpace(msgParts[1])
	}

	newEvent.Command = msgParts[0][len(m.prefix):]

	if newEvent.FromChannel() {
		m.public.HandleEvent(c, newEvent)
	} else {
		m.private.HandleEvent(c, newEvent)
	}
}

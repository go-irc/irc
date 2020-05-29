package irc

// TODO: store all nicks by uuid and map them in outgoing seabird events rather
// than passing the nicks around directly

// TODO: properly handle figuring out the mode when it changes for a user.

import (
	"errors"
	"strings"
	"sync"
)

// Tracker provides a convenient interface to track users, the channels they are
// in, and what modes they have in those channels.
type Tracker struct {
	sync.RWMutex

	channels    map[string]*ChannelState
	isupport    *ISupportTracker
	currentNick string
}

// NewTracker creates a new tracker instance.
func NewTracker(isupport *ISupportTracker) *Tracker {
	return &Tracker{
		channels: make(map[string]*ChannelState),
		isupport: isupport,
	}
}

// ChannelState represents the current state of a channel, including the name,
// topic, and all users in it.
type ChannelState struct {
	Name  string
	Topic string
	Users map[string]struct{}
}

// ListChannels will list the names of all known channels.
func (t *Tracker) ListChannels() []string {
	t.RLock()
	defer t.RUnlock()

	ret := make([]string, 0, len(t.channels))
	for channel := range t.channels {
		ret = append(ret, channel)
	}

	return ret
}

// GetChannel will look up the ChannelState for a given channel name. It will
// return nil if the channel is unknown.
func (t *Tracker) GetChannel(name string) *ChannelState {
	t.RLock()
	defer t.RUnlock()

	return t.channels[name]
}

// Handle needs to be called for all 001, 332, 353, JOIN, TOPIC, PART, KICK,
// QUIT, and NICK messages. All other messages will be ignored. Note that this
// will not handle calling the underlying ISupportTracker's Handle method.
func (t *Tracker) Handle(msg *Message) error {
	switch msg.Command {
	case "001":
		return t.handle001(msg)
	case "332":
		return t.handleRplTopic(msg)
	case "353":
		return t.handleRplNamReply(msg)
	case "JOIN":
		return t.handleJoin(msg)
	case "TOPIC":
		return t.handleTopic(msg)
	case "PART":
		return t.handlePart(msg)
	case "KICK":
		return t.handleKick(msg)
	case "QUIT":
		return t.handleQuit(msg)
	case "NICK":
		return t.handleNick(msg)
	}

	return nil
}

func (t *Tracker) handle001(msg *Message) error {
	if len(msg.Params) != 2 {
		return errors.New("malformed RPL_WELCOME message")
	}

	t.Lock()
	defer t.Unlock()

	t.currentNick = msg.Params[0]

	return nil
}

func (t *Tracker) handleTopic(msg *Message) error {
	if len(msg.Params) != 2 {
		return errors.New("malformed TOPIC message")
	}

	channel := msg.Params[0]
	topic := msg.Trailing()

	t.Lock()
	defer t.Unlock()

	if _, ok := t.channels[channel]; !ok {
		return errors.New("received TOPIC message for unknown channel")
	}

	t.channels[channel].Topic = topic

	return nil
}

func (t *Tracker) handleRplTopic(msg *Message) error {
	if len(msg.Params) != 3 {
		return errors.New("malformed RPL_TOPIC message")
	}

	// client set channel topic to topic

	// client := msg.Params[0]
	channel := msg.Params[1]
	topic := msg.Trailing()

	t.Lock()
	defer t.Unlock()

	if _, ok := t.channels[channel]; !ok {
		return errors.New("received RPL_TOPIC for unknown channel")
	}

	t.channels[channel].Topic = topic

	return nil
}

func (t *Tracker) handleJoin(msg *Message) error {
	if len(msg.Params) != 1 {
		return errors.New("malformed JOIN message")
	}

	// user joined channel
	user := msg.Prefix.Name
	channel := msg.Trailing()

	t.Lock()
	defer t.Unlock()

	_, ok := t.channels[channel]

	if !ok {
		if user != t.currentNick {
			return errors.New("received JOIN message for unknown channel")
		}

		t.channels[channel] = &ChannelState{Name: channel, Users: make(map[string]struct{})}
	}

	state := t.channels[channel]
	state.Users[user] = struct{}{}

	return nil
}

func (t *Tracker) handlePart(msg *Message) error {
	if len(msg.Params) < 1 {
		return errors.New("malformed PART message")
	}

	// user joined channel

	user := msg.Prefix.Name
	channel := msg.Params[0]

	t.Lock()
	defer t.Unlock()

	if _, ok := t.channels[channel]; !ok {
		return errors.New("received PART message for unknown channel")
	}

	// If we left the channel, we can drop the whole thing, otherwise just drop
	// this user from the channel.
	if user == t.currentNick {
		delete(t.channels, channel)
	} else {
		state := t.channels[channel]
		delete(state.Users, user)
	}

	return nil
}

func (t *Tracker) handleKick(msg *Message) error {
	if len(msg.Params) != 3 {
		return errors.New("malformed KICK message")
	}

	// user was kicked from channel by actor

	// actor := msg.Prefix.Name
	user := msg.Params[1]
	channel := msg.Params[0]

	t.Lock()
	defer t.Unlock()

	if _, ok := t.channels[channel]; !ok {
		return errors.New("received KICK message for unknown channel")
	}

	// If we left the channel, we can drop the whole thing, otherwise just drop
	// this user from the channel.
	if user == t.currentNick {
		delete(t.channels, channel)
	} else {
		state := t.channels[channel]
		delete(state.Users, user)
	}

	return nil
}

func (t *Tracker) handleQuit(msg *Message) error {
	if len(msg.Params) != 1 {
		return errors.New("malformed QUIT message")
	}

	// user quit

	user := msg.Prefix.Name

	t.Lock()
	defer t.Unlock()

	for _, state := range t.channels {
		delete(state.Users, user)
	}

	return nil
}

func (t *Tracker) handleNick(msg *Message) error {
	if len(msg.Params) != 1 {
		return errors.New("malformed NICK message")
	}

	// oldUser renamed to newUser

	oldUser := msg.Prefix.Name
	newUser := msg.Params[0]

	t.Lock()
	defer t.Unlock()

	if t.currentNick == oldUser {
		t.currentNick = newUser
	}

	for _, state := range t.channels {
		if _, ok := state.Users[oldUser]; ok {
			delete(state.Users, oldUser)
			state.Users[newUser] = struct{}{}
		}
	}

	return nil
}

func (t *Tracker) handleRplNamReply(msg *Message) error {
	if len(msg.Params) != 4 {
		return errors.New("malformed RPL_NAMREPLY message")
	}

	channel := msg.Params[2]
	users := strings.Split(strings.TrimSpace(msg.Trailing()), " ")

	prefixes, ok := t.isupport.GetPrefixMap()
	if !ok {
		return errors.New("ISupport missing prefix map")
	}

	t.Lock()
	defer t.Unlock()

	if _, ok := t.channels[channel]; !ok {
		return errors.New("received RPL_NAMREPLY message for untracked channel")
	}

	for _, user := range users {
		i := strings.IndexFunc(user, func(r rune) bool {
			_, ok := prefixes[r]
			return !ok
		})

		if i != -1 {
			user = user[i:]
		}

		// The bot user should be added via JOIN
		if user == t.currentNick {
			continue
		}

		state := t.channels[channel]
		state.Users[user] = struct{}{}
	}

	return nil
}

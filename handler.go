package irc

// InputHandler is a handler which can be used to dispatch incoming
// messages.
type InputHandler interface {
	HandleInput(*Conn, *Message)
}

// InputHandlerFunc is used where you only have a function and don't want
// to deal with making a whole struct.
type InputHandlerFunc func(*Conn, *Message)

// HandleInput allows an InputHandlerFunc to work where an
// InputHandler needs to be passed in
func (f InputHandlerFunc) HandleInput(c *Conn, m *Message) {
	f(c, m)
}

// OutputHandler is a handler which can be used to modify messages
// before they are sent to the server.
type OutputHandler interface {
	HandleOutput(*Conn, *Message) []*Message
}

// OutputHandlerFunc is used where you only have a function and don't
// want to deal with making a whole struct.
type OutputHandlerFunc func(*Conn, *Message) []*Message

// HandleOutput allows an OutputHandlerFunc to work where an
// OutputHandler needs to be passed in.
func (f OutputHandlerFunc) HandleOutput(c *Conn, m *Message) []*Message {
	return f(c, m)
}

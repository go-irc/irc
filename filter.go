package irc

// Filter is a simple interface meant for filtering outgoing messages
// on a Client connection
type Filter interface {
	// Filter is called with the client and the message being sent. If
	// the function returns true, the message will not be sent.
	Filter(c *Client, m *Message) bool
}

// FilterFunc is a simple wrapper around a function which allows it to
// be used as a Filter.
type FilterFunc func(c *Client, m *Message) bool

// Filter returns f(c, m)
func (f FilterFunc) Filter(c *Client, m *Message) bool {
	return f(c, m)
}

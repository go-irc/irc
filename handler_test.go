package irc

import "testing"

func TestHandler(t *testing.T) {
	hit := false
	var f HandlerFunc = func(c *Client, m *Message) {
		hit = true
	}

	f.Handle(nil, nil)
	if !hit {
		t.Errorf("HandlerFunc doesn't work correctly as Handler")
	}
}

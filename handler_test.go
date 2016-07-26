package irc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandlerFunc(t *testing.T) {
	t.Parallel()

	hit := false
	var f HandlerFunc = func(c *Client, m *Message) {
		hit = true
	}

	f.Handle(nil, nil)
	assert.True(t, hit, "HandlerFunc doesn't work correctly as Handler")
}

package irc_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gopkg.in/irc.v4"
)

func TestHandlerFunc(t *testing.T) {
	t.Parallel()

	hit := false
	var f irc.HandlerFunc = func(c *irc.Client, m *irc.Message) {
		hit = true
	}

	f.Handle(nil, nil)
	assert.True(t, hit, "HandlerFunc doesn't work correctly as Handler")
}

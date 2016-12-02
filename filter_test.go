package irc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterFunc(t *testing.T) {
	t.Parallel()

	hit := false
	var f FilterFunc = func(c *Client, m *Message) bool {
		hit = true
		return true
	}

	assert.True(t, f.Filter(nil, nil))
	assert.True(t, hit)
}

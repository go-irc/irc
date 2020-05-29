package irc

import (
	"errors"
	"strings"
	"sync"
)

// ISupportTracker tracks the ISUPPORT values returned by servers and provides a
// convenient way to access them.
//
// From http://www.irc.org/tech_docs/draft-brocklesby-irc-isupport-03.txt
//
// 005    RPL_ISUPPORT.
type ISupportTracker struct {
	sync.RWMutex

	data map[string]string
}

// NewISupportTracker creates a new tracker instance with a set of sane defaults
// if the server is missing them.
func NewISupportTracker() *ISupportTracker {
	return &ISupportTracker{
		data: map[string]string{
			"PREFIX": "(ov)@+",
		},
	}
}

// Handle needs to be called for all 005 IRC messages. All other messages will
// be ignored.
func (t *ISupportTracker) Handle(msg *Message) error {
	// Ensure only ISupport messages go through here
	if msg.Command != "005" {
		return nil
	}

	if len(msg.Params) < 2 {
		return errors.New("malformed RPL_ISUPPORT message")
	}

	// Check for really old servers (or servers which based 005 off of rfc2812).
	if !strings.HasSuffix(msg.Trailing(), "server") {
		return errors.New("received invalid RPL_ISUPPORT message")
	}

	t.Lock()
	defer t.Unlock()

	for _, param := range msg.Params[1 : len(msg.Params)-1] {
		data := strings.SplitN(param, "=", 2)
		if len(data) < 2 {
			t.data[data[0]] = ""
			continue
		}

		// TODO: this should properly handle decoding values containing \xHH
		t.data[data[0]] = data[1]
	}

	return nil
}

// IsEnabled will check for boolean ISupport values. Note that for ISupport
// boolean true simply means the value exists.
func (t *ISupportTracker) IsEnabled(key string) bool {
	t.RLock()
	defer t.RUnlock()

	_, ok := t.data[key]
	return ok
}

// GetList will check for list ISupport values.
func (t *ISupportTracker) GetList(key string) ([]string, bool) {
	t.RLock()
	defer t.RUnlock()

	data, ok := t.data[key]
	if !ok {
		return nil, false
	}

	return strings.Split(data, ","), true
}

// GetMap will check for map ISupport values.
func (t *ISupportTracker) GetMap(key string) (map[string]string, bool) {
	t.RLock()
	defer t.RUnlock()

	data, ok := t.data[key]
	if !ok {
		return nil, false
	}

	ret := make(map[string]string)

	for _, v := range strings.Split(data, ",") {
		innerData := strings.SplitN(v, ":", 2)
		if len(innerData) != 2 {
			return nil, false
		}

		ret[innerData[0]] = innerData[1]
	}

	return ret, true
}

// GetRaw will get the raw ISupport values.
func (t *ISupportTracker) GetRaw(key string) (string, bool) {
	t.RLock()
	defer t.RUnlock()

	ret, ok := t.data[key]
	return ret, ok
}

// GetPrefixMap gets the mapping of mode to symbol for the PREFIX value.
// Unfortunately, this is fairly specific, so it can only be used with PREFIX.
func (t *ISupportTracker) GetPrefixMap() (map[rune]rune, bool) {
	// Sample: (qaohv)~&@%+
	prefix, _ := t.GetRaw("PREFIX")

	// We only care about the symbols
	i := strings.IndexByte(prefix, ')')
	if len(prefix) == 0 || prefix[0] != '(' || i < 0 {
		// "Invalid prefix format"
		return nil, false
	}

	// We loop through the string using range so we get bytes, then we throw the
	// two results together in the map.
	symbols := make([]rune, 0, len(prefix)/2-1) // ~&@%+
	for _, r := range prefix[i+1:] {
		symbols = append(symbols, r)
	}

	modes := make([]rune, 0, len(symbols)) // qaohv
	for _, r := range prefix[1:i] {
		modes = append(modes, r)
	}

	if len(modes) != len(symbols) {
		// "Mismatched modes and symbols"
		return nil, false
	}

	prefixes := make(map[rune]rune)
	for k := range symbols {
		prefixes[symbols[k]] = modes[k]
	}

	return prefixes, true
}

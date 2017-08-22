package irc

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestAction is used to execute an action during a stream test. If a
// non-nil error is returned the test will be failed.
type TestAction func(t *testing.T, c *Client, rw *testReadWriter)

func SendLine(output string) TestAction {
	return func(t *testing.T, c *Client, rw *testReadWriter) {
		// First we send the message
		select {
		case rw.readChan <- output:
		case <-rw.exiting:
			assert.Fail(t, "Failed to send")
		}

		// Now we wait for the buffer to be emptied
		select {
		case <-rw.readEmptyChan:
		case <-rw.exiting:
			assert.Fail(t, "Failed to send whole message")
		}
	}
}

func SendFunc(cb func() string) TestAction {
	return func(t *testing.T, c *Client, rw *testReadWriter) {
		SendLine(cb())(t, c, rw)
	}
}

func LineFunc(cb func(m *Message)) TestAction {
	return func(t *testing.T, c *Client, rw *testReadWriter) {
		select {
		case line := <-rw.writeChan:
			cb(MustParseMessage(line))
		case <-time.After(1 * time.Second):
			assert.Fail(t, "LineFunc timeout")
		case <-rw.exiting:
		}
	}
}

func ExpectLine(input string) TestAction {
	return ExpectLineWithTimeout(input, 1*time.Second)
}

func ExpectLineWithTimeout(input string, timeout time.Duration) TestAction {
	return func(t *testing.T, c *Client, rw *testReadWriter) {
		select {
		case line := <-rw.writeChan:
			assert.Equal(t, input, line)
		case <-time.After(timeout):
			assert.Fail(t, "ExpectLine timeout on %s", input)
		case <-rw.exiting:
		}
	}
}

func Delay(delay time.Duration) TestAction {
	return func(t *testing.T, c *Client, rw *testReadWriter) {
		select {
		case <-time.After(delay):
		case <-rw.exiting:
		}
	}
}

type testReadWriter struct {
	actions          []TestAction
	queuedWriteError error
	writeChan        chan string
	queuedReadError  error
	readChan         chan string
	readEmptyChan    chan struct{}
	exiting          chan struct{}
	clientDone       chan struct{}
	serverBuffer     bytes.Buffer
}

func (rw *testReadWriter) Read(buf []byte) (int, error) {
	if rw.queuedReadError != nil {
		err := rw.queuedReadError
		rw.queuedReadError = nil
		return 0, err
	}

	// If there's data left in the buffer, we want to use that first.
	if rw.serverBuffer.Len() > 0 {
		s, err := rw.serverBuffer.Read(buf)
		if err == io.EOF {
			err = nil
		}
		return s, err
	}

	select {
	case rw.readEmptyChan <- struct{}{}:
	default:
	}

	// Read from server. We're either waiting for this whole test to
	// finish or for data to come in from the server buffer. We expect
	// only one read to be happening at once.
	select {
	case data := <-rw.readChan:
		rw.serverBuffer.WriteString(data)
		s, err := rw.serverBuffer.Read(buf)
		if err == io.EOF {
			err = nil
		}
		return s, err
	case <-rw.exiting:
		return 0, io.EOF
	}
}

func (rw *testReadWriter) Write(buf []byte) (int, error) {
	if rw.queuedWriteError != nil {
		err := rw.queuedWriteError
		rw.queuedWriteError = nil
		return 0, err
	}

	// Write to server. We can cheat with this because we know things
	// will be written a line at a time.
	select {
	case rw.writeChan <- string(buf):
		return len(buf), nil
	case <-rw.exiting:
		return 0, errors.New("Connection closed")
	}
}

func runTest(t *testing.T, cc ClientConfig, expectedErr error, actions []TestAction) *Client {
	rw := &testReadWriter{
		actions:       actions,
		writeChan:     make(chan string),
		readChan:      make(chan string),
		readEmptyChan: make(chan struct{}, 1),
		exiting:       make(chan struct{}),
		clientDone:    make(chan struct{}),
	}

	c := NewClient(rw, cc)

	go func() {
		err := c.Run()
		assert.Equal(t, expectedErr, err)
		close(rw.clientDone)
	}()

	// Perform each of the actions
	for _, action := range rw.actions {
		action(t, c, rw)
	}

	// TODO: Make sure there are no more incoming messages

	// Ask everything to shut down and wait for the client to stop.
	close(rw.exiting)
	select {
	case <-rw.clientDone:
	case <-time.After(1 * time.Second):
		assert.Fail(t, "Timeout in client shutdown")
	}

	return c
}

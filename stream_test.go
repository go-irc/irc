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
type TestAction func(t *testing.T, rw *testReadWriter)

func SendLine(output string) TestAction {
	return SendLineWithTimeout(output, 1*time.Second)
}

func AssertClosed() TestAction {
	return func(t *testing.T, rw *testReadWriter) {
		if !rw.closed {
			assert.Fail(t, "Expected conn to be closed")
		}
	}
}

func SendLineWithTimeout(output string, timeout time.Duration) TestAction {
	return func(t *testing.T, rw *testReadWriter) {
		waitChan := time.After(timeout)

		// First we send the message
		select {
		case rw.readChan <- output:
		case <-waitChan:
			assert.Fail(t, "SendLine send timeout on %s", output)
			return
		case <-rw.exiting:
			assert.Fail(t, "Failed to send")
			return
		}

		// Now we wait for the buffer to be emptied
		select {
		case <-rw.readEmptyChan:
		case <-waitChan:
			assert.Fail(t, "SendLine timeout on %s", output)
		case <-rw.exiting:
			assert.Fail(t, "Failed to send whole message")
		}
	}
}

func SendFunc(cb func() string) TestAction {
	return func(t *testing.T, rw *testReadWriter) {
		SendLine(cb())(t, rw)
	}
}

func LineFunc(cb func(m *Message)) TestAction {
	return func(t *testing.T, rw *testReadWriter) {
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
	return func(t *testing.T, rw *testReadWriter) {
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
	return func(t *testing.T, rw *testReadWriter) {
		select {
		case <-time.After(delay):
		case <-rw.exiting:
		}
	}
}

func QueueReadError(err error) TestAction {
	return func(t *testing.T, rw *testReadWriter) {
		select {
		case rw.readErrorChan <- err:
		default:
			assert.Fail(t, "Tried to queue a second read error")
		}
	}
}

func QueueWriteError(err error) TestAction {
	return func(t *testing.T, rw *testReadWriter) {
		select {
		case rw.writeErrorChan <- err:
		default:
			assert.Fail(t, "Tried to queue a second write error")
		}
	}
}

type testReadWriter struct {
	actions        []TestAction
	writeErrorChan chan error
	writeChan      chan string
	readErrorChan  chan error
	readChan       chan string
	readEmptyChan  chan struct{}
	exiting        chan struct{}
	clientDone     chan struct{}
	closed         bool
	serverBuffer   bytes.Buffer
}

func (rw *testReadWriter) maybeBroadcastEmpty() {
	if rw.serverBuffer.Len() == 0 {
		select {
		case rw.readEmptyChan <- struct{}{}:
		default:
		}
	}
}

func (rw *testReadWriter) Read(buf []byte) (int, error) {
	// Check for a read error first
	select {
	case err := <-rw.readErrorChan:
		return 0, err
	default:
	}

	// If there's data left in the buffer, we want to use that first.
	if rw.serverBuffer.Len() > 0 {
		s, err := rw.serverBuffer.Read(buf)
		if err == io.EOF {
			err = nil
		}
		rw.maybeBroadcastEmpty()
		return s, err
	}

	// Read from server. We're waiting for this whole test to finish, data to
	// come in from the server buffer, or for an error. We expect only one read
	// to be happening at once.
	select {
	case err := <-rw.readErrorChan:
		return 0, err
	case data := <-rw.readChan:
		rw.serverBuffer.WriteString(data)
		s, err := rw.serverBuffer.Read(buf)
		if err == io.EOF {
			err = nil
		}
		rw.maybeBroadcastEmpty()
		return s, err
	case <-rw.exiting:
		return 0, io.EOF
	}
}

func (rw *testReadWriter) Write(buf []byte) (int, error) {
	select {
	case err := <-rw.writeErrorChan:
		return 0, err
	default:
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

func (rw *testReadWriter) Close() error {
	select {
	case <-rw.exiting:
		return errors.New("Connection closed")
	default:
		// Ensure no double close
		if !rw.closed {
			rw.closed = true
			close(rw.exiting)
		}
		return nil
	}
}

func newTestReadWriter(actions []TestAction) *testReadWriter {
	return &testReadWriter{
		actions:        actions,
		writeErrorChan: make(chan error, 1),
		writeChan:      make(chan string),
		readErrorChan:  make(chan error, 1),
		readChan:       make(chan string),
		readEmptyChan:  make(chan struct{}, 1),
		exiting:        make(chan struct{}),
		clientDone:     make(chan struct{}),
	}
}

func runClientTest(t *testing.T, cc ClientConfig, expectedErr error, setup func(c *Client), actions []TestAction) *Client {
	rw := newTestReadWriter(actions)
	c := NewClient(rw, cc)

	if setup != nil {
		setup(c)
	}

	go func() {
		err := c.Run()
		assert.Equal(t, expectedErr, err)
		close(rw.clientDone)
	}()

	runTest(t, rw, actions)

	return c
}

func runTest(t *testing.T, rw *testReadWriter, actions []TestAction) {
	// Perform each of the actions
	for _, action := range rw.actions {
		action(t, rw)
	}

	// TODO: Make sure there are no more incoming messages

	// Ask everything to shut down
	rw.Close()

	// Wait for the client to stop
	select {
	case <-rw.clientDone:
	case <-time.After(1 * time.Second):
		assert.Fail(t, "Timeout in client shutdown")
	}
}

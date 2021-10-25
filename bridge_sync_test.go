package opinionatedevents

import (
	"fmt"
	"testing"
	"time"
)

func TestSyncBridge(t *testing.T) {
	t.Run("fails if handler is not pushed", func(t *testing.T) {
		destination := newTestSyncDestination()
		bridge := newSyncBridge(destination)

		if err := bridge.take(NewMessage("test")); err == nil {
			t.Errorf("expected error to not be nil")
		}

		destination.pushHandler(func(_ *Message) error {
			return nil
		})

		if err := bridge.take(NewMessage("test")); err != nil {
			t.Errorf("expected error to be nil")
		}
	})

	t.Run("synchronously handles events", func(t *testing.T) {
		destination := newTestSyncDestination()
		bridge := newSyncBridge(destination)

		countToHandle := 5

		// keep track of how many messages have been handled
		handled := 0

		for i := 0; i < countToHandle; i++ {
			destination.pushHandler(func(_ *Message) error {
				handled += 1
				return nil
			})

			// try to deliver the next message
			if err := bridge.take(NewMessage("test")); err != nil {
				t.Fatal(err)
			}

			expected := i + 1

			if handled != expected {
				t.Errorf("total handled (%d) did not match (%d)", handled, expected)
			}
		}

		if handled != countToHandle {
			t.Errorf("total handled (%d) was not %d", handled, countToHandle)
		}
	})

	t.Run("synchronously waits for slow delivery", func(t *testing.T) {
		destination := newTestSyncDestination()
		bridge := newSyncBridge(destination)

		waitFor := 250

		destination.pushHandler(func(_ *Message) error {
			time.Sleep(time.Duration(waitFor) * time.Millisecond)
			return nil
		})

		startAt := time.Now()
		bridge.take(NewMessage("test"))
		overAt := time.Now()

		if overAt.Sub(startAt).Milliseconds() < int64(waitFor) {
			t.Errorf("bridge returned too fast")
		}
	})

	t.Run("fails if message could not be delivered", func(t *testing.T) {
		destination := newTestSyncDestination()
		bridge := newSyncBridge(destination)

		expectedErr := fmt.Errorf("something went wrong")

		destination.pushHandler(func(_ *Message) error {
			return expectedErr
		})

		if err := bridge.take(NewMessage("test")); err != expectedErr {
			t.Errorf("expected error to not be nil")
		}
	})
}

type testSyncDestination struct {
	handlers []func(message *Message) error
}

func (d *testSyncDestination) deliver(message *Message) error {
	handler, err := d.nextHandler()
	if err != nil {
		return err
	}

	if err := handler(message); err != nil {
		return err
	}

	return nil
}

func (d *testSyncDestination) nextHandler() (func(message *Message) error, error) {
	if len(d.handlers) == 0 {
		return nil, fmt.Errorf("no handlers left")
	}

	handler := d.handlers[0]
	d.handlers = d.handlers[1:]

	return handler, nil
}

func (d *testSyncDestination) pushHandler(handler func(message *Message) error) {
	d.handlers = append(d.handlers, handler)
}

func newTestSyncDestination() *testSyncDestination {
	return &testSyncDestination{
		handlers: []func(message *Message) error{},
	}
}
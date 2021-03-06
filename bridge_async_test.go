package opinionatedevents

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAsyncBridge(t *testing.T) {
	ctx := context.Background()

	t.Run("delivers a message", func(t *testing.T) {
		destination := newTestDestination()
		bridge := newAsyncBridge(3, 500, destination)

		bridge.deliveryConfig.waitBetween = 0

		attempts := 0

		destination.pushHandler(func(_ context.Context, _ *Message) error {
			attempts += 1
			return nil
		})

		msg, err := NewMessage("test.test", nil)
		assert.NoError(t, err)

		envelope := bridge.take(ctx, msg)
		assert.NoError(t, waitForSuccessEnvelope(envelope))

		assert.Equal(t, 1, attempts)
	})

	t.Run("retries delivering a failed message", func(t *testing.T) {
		destination := newTestDestination()
		bridge := newAsyncBridge(3, 500, destination)

		bridge.deliveryConfig.waitBetween = 0

		attempts := 0

		for i := 0; i < 2; i++ {
			isFirstHandler := i == 0

			destination.pushHandler(func(_ context.Context, _ *Message) error {
				attempts += 1

				if isFirstHandler {
					return fmt.Errorf("delivery failed")
				} else {
					return nil
				}
			})
		}

		msg, err := NewMessage("test.test", nil)
		assert.NoError(t, err)

		envelope := bridge.take(ctx, msg)
		assert.NoError(t, waitForSuccessEnvelope(envelope))

		assert.Equal(t, 2, attempts)
	})

	t.Run("retries delivering a failed message only for failed destinations", func(t *testing.T) {
		destination1 := newTestDestination()
		destination2 := newTestDestination()

		bridge := newAsyncBridge(3, 500, destination1, destination2)

		bridge.deliveryConfig.waitBetween = 0

		attempts := []int{0, 0}

		for i, d := range []*testDestination{destination1, destination2} {
			attemptIndex := i
			shouldFail := i == 0

			for u := 0; u < bridge.deliveryConfig.maxAttempts; u += 1 {
				d.pushHandler(func(_ context.Context, _ *Message) error {
					attempts[attemptIndex] = attempts[attemptIndex] + 1

					if shouldFail {
						return fmt.Errorf("delivery failed")
					} else {
						return nil
					}
				})
			}
		}

		msg, err := NewMessage("test.test", nil)
		assert.NoError(t, err)

		envelope := bridge.take(ctx, msg)
		assert.Error(t, waitForSuccessEnvelope(envelope))

		assert.Equal(t, 3, attempts[0])
		assert.Equal(t, 1, attempts[1])
	})

	t.Run("gives up delivering a message after max attempts", func(t *testing.T) {
		destination := newTestDestination()
		bridge := newAsyncBridge(3, 500, destination)

		bridge.deliveryConfig.maxAttempts = 3
		bridge.deliveryConfig.waitBetween = 0

		attempts := 0

		for i := 0; i < bridge.deliveryConfig.maxAttempts+1; i++ {
			destination.pushHandler(func(_ context.Context, _ *Message) error {
				attempts += 1
				return fmt.Errorf("delivery failed")
			})
		}

		msg, err := NewMessage("test.test", nil)
		assert.NoError(t, err)

		envelope := bridge.take(ctx, msg)
		assert.Error(t, waitForSuccessEnvelope(envelope))

		assert.Equal(t, bridge.deliveryConfig.maxAttempts, attempts)
	})

	t.Run("drain waits for slow deliveries", func(t *testing.T) {
		destination := newTestDestination()
		bridge := newAsyncBridge(3, 500, destination)

		waitFor := 100

		destination.pushHandler(func(_ context.Context, _ *Message) error {
			time.Sleep(time.Duration(waitFor) * time.Millisecond)
			return nil
		})

		start := time.Now()

		msg, err := NewMessage("test.test", nil)
		assert.NoError(t, err)

		envelope := bridge.take(ctx, msg)
		assert.NoError(t, waitForSuccessEnvelope(envelope))

		duration := time.Since(start).Milliseconds()

		assert.GreaterOrEqual(t, duration, int64(waitFor))
	})
}

package channel_test

import (
	"sync"
	"testing"
	"time"

	"github.com/gregwebs/go-concurrent/channel"
	"github.com/shoenig/test/must"
)

func TestNewUnboundedChan(t *testing.T) {
	uc := channel.NewUnbounded[int]()
	var zero int

	// Add a value and verify it persists
	uc.Send(42)

	value, ok := uc.Recv()
	must.True(t, ok)
	must.Eq(t, 42, value)

	// Test empty state after receiving the value
	value, ok = uc.TryRecv()
	must.False(t, ok)
	must.Eq(t, zero, value)

	drained := drain(&uc)
	must.Nil(t, drained)
}

func TestUnboundedChanSendRecv(t *testing.T) {
	uc := channel.NewUnbounded[string]()

	// Test single send and receive
	uc.Send("hello")
	value, ok := uc.Recv()
	must.True(t, ok)
	must.Eq(t, "hello", value)

	// Test receive on empty channel
	value, ok = uc.TryRecv()
	must.False(t, ok)
	must.Eq(t, "", value)

	// Test multiple sends and receives
	values := []string{"one", "two", "three"}
	for _, v := range values {
		uc.Send(v)
	}

	for _, expected := range values {
		value, ok := uc.Recv()
		must.True(t, ok)
		must.Eq(t, expected, value)
	}

	// Verify channel is empty
	value, ok = uc.TryRecv()
	must.False(t, ok)
	must.Eq(t, "", value)
}

func TestUnboundedChanDrain(t *testing.T) {
	uc := channel.NewUnbounded[int]()

	// Test drain after sends
	values := []int{1, 2, 3, 4, 5}
	for _, v := range values {
		uc.Send(v)
	}

	drained := drain(&uc)
	must.Len(t, len(values), drained)
	for i, expected := range values {
		must.Eq(t, expected, drained[i])
	}

	// Verify channel is empty after drain
	value, ok := uc.TryRecv()
	must.False(t, ok)
	must.Eq(t, 0, value)
}

func TestUnboundedChanRace(t *testing.T) {
	uc := channel.NewUnbounded[int]()
	const numGoroutines = 5
	const numOperations = 20

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // Half sending, half receiving

	// Start senders
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				uc.Send(id*1000 + j)
				time.Sleep(time.Microsecond) // Small delay to allow interleaving
			}
		}(i)
	}

	// Track received values
	receivedValues := channel.NewUnbounded[int]()

	// Start receivers
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				// Try to receive a few times with small delays
				for attempt := 0; attempt < 10; attempt++ {
					if val, ok := uc.Recv(); ok {
						receivedValues.Send(val)
						break
					}
					time.Sleep(time.Millisecond)
				}
			}
		}()
	}

	// Wait for all goroutines to finish
	wg.Wait()

	for _, v := range uc.CloseWithOverflow() {
		receivedValues.Send(v)
	}

	// We should have received at least some values
	// We can't guarantee exactly how many due to the concurrent nature
	must.True(t, len(drain(&receivedValues)) > 0)
}

func BenchmarkUnboundedChanSendRecv(b *testing.B) {
	uc := channel.NewUnbounded[int]()
	b.ResetTimer()
	b.Run("Send", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			uc.Send(i)
		}
	})

	b.Run("Recv", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			uc.Recv()
		}
	})
}

func BenchmarkUnboundedChanDrain(b *testing.B) {
	b.Run("SendThenDrain", func(b *testing.B) {
		uc := channel.NewUnbounded[int]()
		for i := 0; i < b.N; i++ {
			uc.Send(i)
		}
		b.ResetTimer()
		drain(&uc)
	})
}

func drain[T any](uc *channel.Unbounded[T]) []T {
	items := uc.CloseWithOverflow()
	for {
		if x, received := uc.TryRecv(); !received {
			break
		} else {
			items = append(items, x)
		}
	}
	return items
}

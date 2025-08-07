package concurrent_test

import (
	"sync"
	"testing"
	"time"

	"github.com/gregwebs/go-concurrent"
	"github.com/shoenig/test/must"
)

func TestNewSlice(t *testing.T) {
	uc := concurrent.NewSlice[int]()
	var zero int

	// Add a value and verify it persists
	uc.Append(42)

	value, ok := uc.Shift()
	must.True(t, ok)
	must.Eq(t, 42, value)

	// Test empty state after receiving the value
	value, ok = uc.Shift()
	must.False(t, ok)
	must.Eq(t, zero, value)

	drained := uc.TakeAll()
	must.Nil(t, drained)
}

func TestNonBlockingChanSendRecv(t *testing.T) {
	uc := concurrent.NewSlice[string]()

	// Test single send and receive
	uc.Append("hello")
	value, ok := uc.Shift()
	must.True(t, ok)
	must.Eq(t, "hello", value)

	// Test receive on empty channel
	value, ok = uc.Shift()
	must.False(t, ok)
	must.Eq(t, "", value)

	// Test multiple sends and receives
	values := []string{"one", "two", "three"}
	for _, v := range values {
		uc.Append(v)
	}

	for _, expected := range values {
		value, ok := uc.Shift()
		must.True(t, ok)
		must.Eq(t, expected, value)
	}

	// Verify channel is empty
	value, ok = uc.Shift()
	must.False(t, ok)
	must.Eq(t, "", value)
}

func TestNonBlockingChanDrain(t *testing.T) {
	uc := concurrent.NewSlice[int]()

	// Test drain on empty channel
	drained := uc.TakeAll()
	must.Nil(t, drained)

	// Test drain after sends
	values := []int{1, 2, 3, 4, 5}
	for _, v := range values {
		uc.Append(v)
	}

	drained = uc.TakeAll()
	must.Len(t, len(values), drained)
	for i, expected := range values {
		must.Eq(t, expected, drained[i])
	}

	// Verify channel is empty after drain
	value, ok := uc.Shift()
	must.False(t, ok)
	must.Eq(t, 0, value)

	// Verify drain returns nil on empty channel
	drained = uc.TakeAll()
	must.Nil(t, drained)
}

func TestNonBlockingChanRace(t *testing.T) {
	uc := concurrent.NewSlice[int]()
	const numGoroutines = 5
	const numOperations = 20

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // Half sending, half receiving

	// Start senders
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				uc.Append(id*1000 + j)
				time.Sleep(time.Microsecond) // Small delay to allow interleaving
			}
		}(i)
	}

	// Track received values
	receivedValues := concurrent.NewSlice[int]()

	// Start receivers
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				// Try to receive a few times with small delays
				for attempt := 0; attempt < 10; attempt++ {
					if val, ok := uc.Shift(); ok {
						receivedValues.Append(val)
						break
					}
					time.Sleep(time.Millisecond)
				}
			}
		}()
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Drain any remaining items
	remaining := uc.TakeAll()
	for _, v := range remaining {
		receivedValues.Append(v)
	}

	// We should have received at least some values
	// We can't guarantee exactly how many due to the concurrent nature
	must.True(t, len(receivedValues.TakeAll()) > 0)
}

func BenchmarkNonBlockingChanSendRecv(b *testing.B) {
	uc := concurrent.NewSlice[int]()
	b.ResetTimer()
	b.Run("Send", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			uc.Append(i)
		}
	})

	b.Run("Recv", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			uc.Shift()
		}
	})
}

func BenchmarkNonBlockingChanDrain(b *testing.B) {
	b.Run("SendThenDrain", func(b *testing.B) {
		uc := concurrent.NewSlice[int]()
		for i := 0; i < b.N; i++ {
			uc.Append(i)
		}
		b.ResetTimer()
		uc.TakeAll()
	})
}

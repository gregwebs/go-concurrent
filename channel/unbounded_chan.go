package channel

import (
	"sync"

	"github.com/gregwebs/go-concurrent"
)

// Unbounded is a channel with an additional overflowing buffer
// It presents a channel-like API with Send and Recv methods.
// The Send method is non-blocking.
// It also allows access to the channel itself with Receiver()
type Unbounded[T any] struct {
	buffer    []T
	channel   chan T
	receivers []chan T
	m         *sync.Mutex
}

// Send is non-blocking
func (ub *Unbounded[T]) Send(x T) {
	ub.m.Lock()
	defer ub.m.Unlock()
	if ub.transferBufferToChannel() && concurrent.TrySend(ub.channel, x) {
		return
	}
	if ub.buffer == nil {
		ub.buffer = make([]T, 0, chanSize)
	}
	ub.buffer = append(ub.buffer, x)
}

// Recv is blocking
func (ub *Unbounded[T]) Recv() (T, bool) {
	ub.transferBufferToChannelLocked()
	x, received := <-ub.channel
	return x, received
}

// Non-blocking receive
// return zero, false if there is no data
func (ub *Unbounded[T]) TryRecv() (T, bool) {
	ub.transferBufferToChannelLocked()
	return concurrent.TryRecv(ub.channel)
}

// Receiver returns a receivable channel.
// That channel will receive data from the UnboundedChan.
// Multiple channels can be created and they will receive different data.
func (ub *Unbounded[T]) Receiver() <-chan T {
	receiver := make(chan T, chanSize)
	func() {
		ub.m.Lock()
		defer ub.m.Unlock()
		if ub.receivers == nil {
			ub.receivers = make([]chan T, 0)
		}
		ub.receivers = append(ub.receivers, receiver)
	}()
	go func() {
		for x := range ub.channel {
			receiver <- x
			ub.transferBufferToChannel()
		}
	}()
	return receiver
}

func (ub Unbounded[T]) Len() int {
	return len(ub.buffer) + len(ub.channel)
}

// CloseWithOverflow closes the underlying channel
// Any additional overflow buffer data that can not be written to the channel is returned
func (ub *Unbounded[T]) CloseWithOverflow() []T {
	ub.m.Lock()
	defer ub.m.Unlock()
	var overflow []T
	if len(ub.buffer) > 0 {
		overflow = ub.buffer
		ub.buffer = []T{}
	}
	close(ub.channel)
	for _, recv := range ub.receivers {
		close(recv)
	}
	ub.receivers = nil
	return overflow
}

func (ub *Unbounded[T]) transferBufferToChannelLocked() {
	ub.m.Lock()
	defer ub.m.Unlock()
	ub.transferBufferToChannel()
}

// The caller must lock and unlock.
// transferBufferToChannelLocked can be used for locking.
// return false if could not send to the channel
func (ub *Unbounded[T]) transferBufferToChannel() bool {
	if len(ub.buffer) == 0 {
		return true
	}
	for {
		if concurrent.TrySend(ub.channel, ub.buffer[0]) {
			ub.buffer = ub.buffer[1:]
			if len(ub.buffer) == 0 {
				return true
			}
		} else {
			return false
		}
	}
}

var chanSize = 100

// NewUnbounded create an [Unbounded].
func NewUnbounded[T any]() Unbounded[T] {
	return Unbounded[T]{
		channel: make(chan T, chanSize),
		m:       &sync.Mutex{},
	}
}

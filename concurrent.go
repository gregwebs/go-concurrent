package concurrent

import (
	"sync"

	"github.com/gregwebs/errors"
	"github.com/gregwebs/go-recovery"
)

// Concurrent spawns n go routines each of which runs the given function.
// a panic in the given function is recovered and converted to an error
// errors are returned as []error, a slice of errors
// If there are no errors, the slice will be nil
// To combine the errors as a single error, use errors.Join
func GoN(n int, fn func(int) error) []error {
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		i := i
		wg.Add(1)
		go recovery.GoHandler(func(err error) { errs[i] = err }, func() error {
			defer wg.Done()
			errs[i] = fn(i)
			return nil
		})
	}
	wg.Wait()
	return errors.Joins(errs...)
}

// Merge multiple channels together.
// From this article: https://go.dev/blog/pipelines
func ChannelMerge[T any](cs ...<-chan T) <-chan T {
	var wg sync.WaitGroup
	out := make(chan T)

	// Start an output goroutine for each input channel in cs.  output
	// copies values from c to out until c is closed, then calls wg.Done.
	output := func(c <-chan T) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}
	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	// Start a goroutine to close out once all the output goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

// try to receive from a channel, return false if nothing received
func TryRecv[T any](c <-chan T) (receivedObject T, received bool) {
	select {
	case receivedObject = <-c:
		received = true
	default:
		received = false
	}
	return
}

// try to send to a channel, return true if sent, false if not
func TrySend[T any](c chan<- T, obj T) bool {
	select {
	case c <- obj:
		return true
	default:
		return false
	}
}

// UnboundedChan transfers its contents into an unbounded slice
// Close the channel and retrieve the slice data with Drain()
type UnboundedChan[T any] struct {
	chanT  chan T
	sliceT []T
	done   chan struct{}
}

func (uc UnboundedChan[T]) Send(x T) {
	uc.chanT <- x
}

func (uc UnboundedChan[T]) Drain() []T {
	close(uc.chanT)
	<-uc.done
	return uc.sliceT
}

func NewUnboundedChan[T any]() UnboundedChan[T] {
	chanSize := 10
	uc := UnboundedChan[T]{
		chanT:  make(chan T, chanSize),
		sliceT: make([]T, 0, chanSize),
		done:   make(chan struct{}),
	}
	go func() {
		defer func() {
			uc.done <- struct{}{}
		}()
		for x := range uc.chanT {
			uc.sliceT = append(uc.sliceT, x)
		}
	}()
	return uc
}

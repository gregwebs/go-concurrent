package concurrent

import (
	"sync"

	"github.com/gregwebs/go-recovery"
)

// GoN runs a function in parallel multiple times using n goroutines.
//
// It recovers any panics that occur during the execution of the function
// and returns them as a slice of errors. If no errors occurred, nil will be returned.
//
// Use [errors.Join] to combine the individual errors into a single error.
func GoN(n int, fn func(int) error) []error {
	return GoConcurrent().GoN(n, fn)
}

// GoEach runs a go routine for each item in an Array.
// It is a convenient generic wrapper around [GoN].
//
// It recovers any panics that occur during the execution of the function
// and returns them as a slice of errors. If no errors occurred, nil will be returned.
//
// Use [errors.Join] to combine the individual errors into a single error.
func GoEach[T any](all []T, fn func(T) error) []error {
	return GoN(len(all), func(n int) error {
		item := all[n]
		return fn(item)
	})
}

// [GoConcurrent] is the default implementation for launching a routine.
// It just uses the `go` keyword.
func GoConcurrent() GoRoutine {
	return GoRoutine(func(work func()) { go work() })
}

// [GoSerial] allows for running in serial for debugging
func GoSerial() GoRoutine {
	return GoRoutine(func(work func()) { work() })
}

// GoRoutine allows for inserting hooks before launching Go routines
// [GoConcurrent] is the default implementation.
// [GoSerial] allows for running in serial for debugging
type GoRoutine func(func())

// The same as [GoN] but with go routine launching configured by a GoRoutine.
func (gr GoRoutine) GoN(n int, fn func(int) error) []error {
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		i := i
		wg.Add(1)
		gr(func() {
			recovery.GoHandler(func(err error) { errs[i] = err }, func() error {
				defer wg.Done()
				errs[i] = fn(i)
				return nil
			})
		})
	}
	wg.Wait()
	return joins(errs...)
}

// The same as [GoEach] but with go routine launching configured by a GoRoutine.
//
// [GoEach] uses generics, so it cannot be called directly as a method.
// Instead, apply the [GoEach] arguments first, than apply the [GoRoutine] to the resulting function.
func GoEachRoutine[T any](all []T, work func(T) error) func(gr GoRoutine) []error {
	return func(gr GoRoutine) []error {
		return gr.GoN(len(all), func(n int) error {
			item := all[n]
			return work(item)
		})
	}
}

// ChannelMerge merges multiple channels together.
// See the article [Go Concurrency Patterns].
//
// [Go Concurrency Patterns]: https://go.dev/blog/pipelines
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

// TryRecv preforms a non-blocking receive from a channel.
// It returns false if nothing received.
func TryRecv[T any](c <-chan T) (receivedObject T, received bool) {
	select {
	case receivedObject = <-c:
		received = true
	default:
		received = false
	}
	return
}

// TrySend performs a non-blocking send to a channel.
// It returns true if sent, false if not
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

// NewUnboundedChan create an UnboundedChan that transfers its contents into an unbounded slice
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

func joins(errs ...error) []error {
	n := 0
	for _, err := range errs {
		if err != nil {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	newErrs := make([]error, 0, n)
	for _, err := range errs {
		if err != nil {
			newErrs = append(newErrs, err)
		}
	}
	return newErrs
}

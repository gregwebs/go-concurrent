package concurrent

import (
	"fmt"
	"sync"
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
			err := recovered(func() error {
				defer wg.Done()
				errs[i] = fn(i)
				return nil
			})
			if err != nil {
				errs[i] = err
			}
		})
	}
	wg.Wait()
	return joins(errs...)
}

func recovered(fn func() error) (err error) {
	defer func() {
		r := recover()
		if r != nil {
			if re, ok := r.(error); ok {
				err = re
			} else {
				err = fmt.Errorf("panic: %v", r)
			}
		}
	}()
	return fn()
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

// TryRecv preforms a non-blocking receive from a channel.
// It returns false if nothing received.
func TryRecv[T any](c <-chan T) (receivedObject T, received bool) {
	select {
	case receivedObject, received = <-c:
	default:
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

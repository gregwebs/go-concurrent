# go-concurrent

Go library to run code concurrently

For full documentation see the [GoDoc](https://pkg.go.dev/github.com/gregwebs/go-concurrent).

## Running multiple Go routines concurrency

* GoN - run N go routines concurrently
* GoEach - run a go routine for each array element
* Group - Similar to x/sync/errgroup but
  * catches panics (can be disabled)
  * by default returns all errors
  * can fail fast and return the first error without waiting for all go routines to complete

It is possible to instrument how the go routines are launched and panics are handled.
See:

* GoRoutine - create your own go routine launcher
  * Serial() - running in serial for debugging
  * RaisePanics() - raise panics rather than trapping them
  * GoErrHandler(errHandler func(error), work func() error) - launch a go routine with a configured error handler

* GoRoutine.GoN(...)
* GoEachRoutine(...)(GoRoutine)
* Group.SetGoRoutine(GoRoutine)

## General concurrency helpers exposed

* TrySend
* TryRecv
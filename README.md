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

It is possible to instrument how the go routines are launched or launch them in serial for debugging.
See:

* GoSerial - running in serial for debugging
* GoRoutine - create your own go routine launcher
  * SetWrapFn(nil) - disable panic catching
  * SetGo(func(func())) - add hooks to go routine launching
  * LaunchGoRoutine(func(func())) - launch a go routine with configured hooks

* GoRoutine.GoN(...)
* GoEachRoutine(...)(GoRoutine)
* Group.SetGoRoutine(GoRoutine)

## General concurrency helpers exposed

* UnboundedChan
* ChannelMerge
* TrySend
* TryRecv

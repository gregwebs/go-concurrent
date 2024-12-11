# go-concurrent

Go library to run code concurrently

## Running multiple Go routines concurrency

* GoN - run N go routines concurrently
* GoEach - run a go routine for each array element
* NewGroupContext - Similar to x/sync/errgroup but catches panics and returns all errors

It is possible to instrument how the go routines are launched or launch them in serial for debugging.
See:

* GoSerial - running in serial for debugging
* GoRoutine - create your own go routine launcher
* GoRoutine.GoN(...)
* GoEachRoutine(...)(GoRoutine)
* group.SetGoRoutine(GoRoutine)

## Concurrency helpers

* UnboundedChan
* ChannelMerge
* TrySend
* TryRecv
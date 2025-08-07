package concurrent

import "sync"

var chanSize = 100

// A concurrency safe slice that locks a Mutex before performing operations on the slice.
type slice[T any] struct {
	sliceT []T
	m      *sync.RWMutex
}

func (ls *slice[T]) Append(x T) {
	ls.m.Lock()
	defer ls.m.Unlock()
	if ls.sliceT == nil {
		ls.sliceT = make([]T, 0, chanSize)
	}
	ls.sliceT = append(ls.sliceT, x)
}

func (ls *slice[T]) Shift() (T, bool) {
	ls.m.Lock()
	defer ls.m.Unlock()
	if len(ls.sliceT) > 0 {
		x := ls.sliceT[0]
		ls.sliceT = ls.sliceT[1:]
		return x, true
	}
	var zero T
	return zero, false
}

func (ls *slice[T]) TakeAll() []T {
	ls.m.Lock()
	defer ls.m.Unlock()
	if len(ls.sliceT) == 0 {
		return nil
	}
	result := ls.sliceT
	ls.sliceT = nil
	return result
}

func (ls slice[T]) Get(i int) T {
	ls.m.RLock()
	defer ls.m.RUnlock()
	return ls.sliceT[i]
}

func (ls slice[T]) Len() int {
	ls.m.RLock()
	defer ls.m.RUnlock()
	return len(ls.sliceT)
}

func NewSlice[T any]() slice[T] {
	return slice[T]{
		m: &sync.RWMutex{},
	}
}

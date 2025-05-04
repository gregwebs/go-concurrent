package concurrent

import "sync"

// UnboundedChan presents a channel like API with Send and Recv
// It also provides a Drain function to retrieve all data at once.
type UnboundedChan[T any] struct {
	sliceT []T
	m      *sync.Mutex
}

func (uc *UnboundedChan[T]) Send(x T) {
	uc.m.Lock()
	defer uc.m.Unlock()
	uc.sliceT = append(uc.sliceT, x)
}

// Recv is non-blocking
// If there is no data, it returns a zero value and false
func (uc *UnboundedChan[T]) Recv() (T, bool) {
	uc.m.Lock()
	defer uc.m.Unlock()

	if len(uc.sliceT) == 0 {
		var zero T
		return zero, false
	}

	data := uc.sliceT[0]
	uc.sliceT = uc.sliceT[1:]
	return data, true
}

func (uc *UnboundedChan[T]) Drain() []T {
	uc.m.Lock()
	defer uc.m.Unlock()

	if len(uc.sliceT) == 0 {
		return nil
	}

	data := uc.sliceT
	uc.sliceT = make([]T, 0, len(data))
	return data
}

func (uc UnboundedChan[T]) Len() int {
	uc.m.Lock()
	defer uc.m.Unlock()
	return len(uc.sliceT)
}

// NewUnboundedChan create an UnboundedChan that transfers its contents into an unbounded slice
func NewUnboundedChan[T any]() UnboundedChan[T] {
	chanSize := 100
	uc := UnboundedChan[T]{
		sliceT: make([]T, 0, chanSize),
		m:      &sync.Mutex{},
	}
	return uc
}

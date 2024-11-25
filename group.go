/*
This code is modified from the Go distribution x/sync/errgroup.go
Below is the Go copyright notice

Copyright 2009 The Go Authors.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

  - Redistributions of source code must retain the above copyright

notice, this list of conditions and the following disclaimer.
  - Redistributions in binary form must reproduce the above

copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
  - Neither the name of Google LLC nor the names of its

contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

package concurrent

import (
	"context"
	"fmt"
	"sync"

	"github.com/gregwebs/errors"
	"github.com/gregwebs/go-recovery"
)

type token struct{}

// Similar to x/sync/errgroup. Differences:
// * Wait() will return a slice of all errors encountered.
// * panics in the functions that are ran are recovered and converted to errors.
// Must be constructed with NewGroupContext
type group struct {
	errChan UnboundedChan[error]
	wg      sync.WaitGroup
	cancel  func(error)
	sem     chan token
}

func (g *group) do(fn func() error) {
	g.wg.Add(1)
	go recovery.GoHandler(func(err error) { g.errChan.Send(err) }, func() error {
		defer g.done()
		if err := fn(); err != nil {
			g.errChan.Send(err)
			g.cancel(err)
		}
		return nil
	})
}

func (g *group) done() {
	if g.sem != nil {
		<-g.sem
	}
	g.wg.Done()
}

// Wait waits for any outstanding go routines and returns their errors
// If go routines are started during this Wait,
// their errors might not show up until the next Wait
func (g *group) Wait() []error {
	g.wg.Wait()
	prevErrChan := g.errChan
	g.errChan = NewUnboundedChan[error]()
	errs := prevErrChan.Drain()
	if g.cancel != nil {
		g.cancel(errors.Join(errs...))
	}
	return errors.Joins(errs...)
}

func NewGroupContext(ctx context.Context) (*group, context.Context) {
	ctx, cancel := context.WithCancelCause(ctx)
	return &group{
		cancel:  cancel,
		errChan: NewUnboundedChan[error](),
	}, ctx
}

func (g *group) Go(fn func() error) {
	if g.sem != nil {
		g.sem <- token{}
	}
	g.do(fn)
}

func (g *group) TryGo(fn func() error) bool {
	if g.sem != nil {
		select {
		case g.sem <- token{}:
			// Note: this allows barging iff channels in general allow barging.
		default:
			return false
		}
	}
	g.do(fn)
	return true
}

func (g *group) SetLimit(n int) {
	if n < 0 {
		g.sem = nil
		return
	}
	if len(g.sem) != 0 {
		panic(fmt.Errorf("errgroup: modify limit while %v goroutines in the group are still active", len(g.sem)))
	}
	g.sem = make(chan token, n)
}

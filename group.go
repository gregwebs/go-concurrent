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
	"errors"
	"fmt"
	"sync"
)

type token struct{}

// Group is similar to [x/sync/errgroup].
// Improvements:
//   - Wait() will return a slice of all errors encountered.
//   - panics in the functions that are ran are recovered and converted to errors.
//   - Go routine launching can be configured with [*Group.SetGoRoutine]
//
// Must be constructed with [NewGroupContext]
type Group struct {
	errChan    UnboundedChan[error]
	wg         sync.WaitGroup
	cancel     func(error)
	limiter    chan token
	goRoutine  GoRoutine
	firstError chan error
}

func (g *Group) do(fn func() error) {
	g.wg.Add(1)
	g.goRoutine(func() {
		go func() {
			defer g.done()
			err := recovered(func() error {
				if err := fn(); err != nil {
					g.error(err)
				}
				return nil
			})
			if err != nil {
				g.error(err)
			}
		}()
	})
}

func (g *Group) done() {
	if g.limiter != nil {
		<-g.limiter
	}
	g.wg.Done()
}

func (g *Group) error(err error) {
	if err == nil {
		return
	}
	g.errChan.Send(err)
	if g.firstError == nil {
		g.firstError = make(chan error, 1)
	}
	TrySend(g.firstError, err)
}

// WaitOrError will wait until any go routine returns an error.
// If the error returned is nil then all go routines have completed without error.
// Once a go routine returns an error, that will be returned here as a non-nil error.
// If an error is returned, the caller can call 'Wait' to wait for all go routines to complete.
func (g *Group) WaitOrError() error {
	var err error
	defer func() { g.cancel(err) }()
	err = func() error {
		if g.firstError == nil {
			g.firstError = make(chan error, 1)
		}
		if err, received := g.errChan.Recv(); received {
			return err
		}

		completed := make(chan token)
		go func() {
			g.wg.Wait()
			completed <- token{}
		}()

		select {
		case err := <-g.firstError:
			return err
		case <-completed:
			// Favor firstError over completed: it should have the first error
			if err, received := TryRecv(g.firstError); received {
				return err
			}

			if errs := g.errChan.Drain(); len(errs) > 0 {
				return errs[0]
			}
			return nil
		}
	}()
	return err
}

// Wait waits for any outstanding go routines and returns their errors
// If go routines are started during this Wait,
// their errors might not show up until the next Wait
func (g *Group) Wait() []error {
	var errs []error
	defer func() { g.cancel(errors.Join(errs...)) }()
	g.wg.Wait()
	errs = g.errChan.Drain()
	return joins(errs...)
}

// NewGroupContext constructs a [Group] similar to [x/sync/errgroup] but with enhancements.
// See [Group].
func NewGroupContext(ctx context.Context) (*Group, context.Context) {
	ctx, cancel := context.WithCancelCause(ctx)
	return &Group{
		cancel:    cancel,
		errChan:   NewUnboundedChan[error](),
		goRoutine: GoConcurrent(),
	}, ctx
}

// SetGoRoutine allows configuring how go routines are launched
func (g *Group) SetGoRoutine(gr GoRoutine) {
	g.goRoutine = gr
}

func (g *Group) Go(fn func() error) {
	if g.limiter != nil {
		g.limiter <- token{}
	}
	g.do(fn)
}

func (g *Group) TryGo(fn func() error) bool {
	if g.limiter != nil {
		select {
		case g.limiter <- token{}:
			// Note: this allows barging iff channels in general allow barging.
		default:
			return false
		}
	}
	g.do(fn)
	return true
}

func (g *Group) SetLimit(n int) {
	if n < 0 {
		g.limiter = nil
		return
	}
	if len(g.limiter) != 0 {
		panic(fmt.Errorf("errgroup: modify limit while %v goroutines in the group are still active", len(g.limiter)))
	}
	g.limiter = make(chan token, n)
}

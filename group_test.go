/*
This code is modified from the Go distribution x/sync/errgroup_test.go
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

package concurrent_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gregwebs/go-concurrent"
)

func TestZeroGroup(t *testing.T) {
	err1 := errors.New("errgroup_test: 1")
	err2 := errors.New("errgroup_test: 2")

	cases := []struct {
		errs []error
		name string
	}{
		{errs: []error{}, name: "empty"},
		{errs: []error{nil}, name: "nil"},
		{errs: []error{err1}, name: "single error"},
		{errs: []error{err1, nil}, name: "error and nil"},
		{errs: []error{err1, nil, err2}, name: "error, nil, error"},
	}

	for _, tc := range cases {
		g, ctx := concurrent.NewGroupContext(context.Background())
		gwe, _ := concurrent.NewGroupContext(context.Background())

		for _, err := range tc.errs {
			err := err
			g.Go(func() error { return err })
			gwe.Go(func() error { return err })
		}

		nonNilErrs := joins(tc.errs...)
		{
			gErr := g.Wait()
			if len(gErr) != len(nonNilErrs) {
				t.Errorf(tc.name+": after %T.Go(func() error { return err }) for err in %v\n"+
					"g.Wait() = %v; want %v",
					g, tc.errs, gErr, nonNilErrs)
			}

			canceled := false
			select {
			case <-ctx.Done():
				canceled = true
			default:
			}
			if !canceled {
				t.Errorf("after %T.Go(func() error { return err }) for err in %v\n"+
					"ctx.Done() was not closed",
					g, tc.errs)
			}
		}

		{
			gweErr := g.WaitOrError()
			var nonNilErr error
			if !((gweErr == nil && len(nonNilErrs) == 0) || slices.Contains(nonNilErrs, gweErr)) {
				t.Errorf(tc.name+": after %T.Go(func() error { return err }) for err in %v\n"+
					"g.WaitOrError() = %v; want %v",
					gwe, tc.errs, gweErr, nonNilErr)
			}
		}
	}
}

func TestTryGo(t *testing.T) {
	g, _ := concurrent.NewGroupContext(context.Background())
	n := 42
	g.SetLimit(42)
	ch := make(chan struct{})
	fn := func() error {
		ch <- struct{}{}
		return nil
	}
	for i := 0; i < n; i++ {
		if !g.TryGo(fn) {
			t.Fatalf("TryGo should succeed but got fail at %d-th call.", i)
		}
	}
	if g.TryGo(fn) {
		t.Fatalf("TryGo is expected to fail but succeeded.")
	}
	go func() {
		for i := 0; i < n; i++ {
			<-ch
		}
	}()
	g.Wait()

	if !g.TryGo(fn) {
		t.Fatalf("TryGo should success but got fail after all goroutines.")
	}
	go func() { <-ch }()
	g.Wait()

	// Switch limit.
	g.SetLimit(1)
	if !g.TryGo(fn) {
		t.Fatalf("TryGo should success but got failed.")
	}
	if g.TryGo(fn) {
		t.Fatalf("TryGo should fail but succeeded.")
	}
	go func() { <-ch }()
	g.Wait()

	// Block all calls.
	g.SetLimit(0)
	for i := 0; i < 1<<10; i++ {
		if g.TryGo(fn) {
			t.Fatalf("TryGo should fail but got succeded.")
		}
	}
	g.Wait()
}

func TestGoLimit(t *testing.T) {
	const limit = 10

	g, _ := concurrent.NewGroupContext(context.Background())
	g.SetLimit(limit)
	var active int32
	for i := 0; i <= 1<<10; i++ {
		g.Go(func() error {
			n := atomic.AddInt32(&active, 1)
			if n > limit {
				return fmt.Errorf("saw %d active goroutines; want ≤ %d", n, limit)
			}
			time.Sleep(1 * time.Microsecond) // Give other goroutines a chance to increment active.
			atomic.AddInt32(&active, -1)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkGo(b *testing.B) {
	fn := func() {}
	g, _ := concurrent.NewGroupContext(context.Background())
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		g.Go(func() error { fn(); return nil })
	}
	g.Wait()
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

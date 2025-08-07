package concurrent_test

import (
	"context"
	"testing"

	"github.com/gregwebs/go-concurrent"
	"github.com/gregwebs/go-concurrent/channel"
	"github.com/shoenig/test/must"
)

func TestGoN(t *testing.T) {
	var err []error
	workNone := func(_ int) error { return nil }
	err = concurrent.GoN(0, workNone)
	must.Nil(t, err)
	err = concurrent.GoN(2, workNone)
	must.Nil(t, err)

	tracked := channel.NewUnbounded[bool]()
	workTracked := func(i int) error { tracked.Send(true); return nil }
	err = concurrent.GoN(0, workTracked)
	must.Nil(t, err)
	must.Length(t, 0, tracked)

	tracked = channel.NewUnbounded[bool]()
	err = concurrent.GoN(2, workTracked)
	must.Nil(t, err)
	must.Length(t, 2, tracked)
	r, ok := tracked.Recv()
	must.True(t, ok)
	must.True(t, r)
	r, ok = tracked.Recv()
	must.True(t, ok)
	must.True(t, r)
}

func TestGoNSerials(t *testing.T) {
	var err []error
	gr := concurrent.GoRoutine{}.Serial()
	workNone := func(_ int) error { return nil }
	err = gr.GoN(0, workNone)
	must.Nil(t, err)
	err = gr.GoN(2, workNone)
	must.Nil(t, err)

	tracked := channel.NewUnbounded[bool]()
	workTracked := func(i int) error { tracked.Send(true); return nil }
	err = gr.GoN(0, workTracked)
	must.Nil(t, err)
	must.Length(t, 0, tracked)

	tracked = channel.NewUnbounded[bool]()
	err = gr.GoN(2, workTracked)
	must.Nil(t, err)
	must.Length(t, 2, tracked)
	r, ok := tracked.Recv()
	must.True(t, ok)
	must.True(t, r)
	r, ok = tracked.Recv()
	must.True(t, ok)
	must.True(t, r)
}

func TestGoEach(t *testing.T) {
	var err []error
	items := make([]bool, 10)
	workNone := func(_ bool) error { return nil }
	err = concurrent.GoEach(items, workNone)
	must.Nil(t, err)

	tracked := channel.NewUnbounded[bool]()
	workTracked := func(_ bool) error { tracked.Send(true); return nil }
	err = concurrent.GoEach(items, workTracked)
	must.Nil(t, err)
	must.Length(t, 10, tracked)
	r, ok := tracked.Recv()
	must.True(t, ok)
	must.True(t, r)

	tracked = channel.NewUnbounded[bool]()
	workTracked = func(_ bool) error { tracked.Send(true); return nil }
	err = concurrent.GoEach(items, workTracked)
	must.Nil(t, err)
	must.Length(t, 10, tracked)
	r, ok = tracked.Recv()
	must.True(t, ok)
	must.True(t, r)
}

func TestGoEachSerial(t *testing.T) {
	var err []error
	items := make([]bool, 10)
	workNone := func(_ bool) error { return nil }
	gr := concurrent.GoRoutine{}.Serial()
	err = concurrent.GoEachRoutine(items, workNone)(gr)
	must.Nil(t, err)

	tracked := channel.NewUnbounded[bool]()
	workTracked := func(_ bool) error { tracked.Send(true); return nil }
	err = concurrent.GoEachRoutine(items, workTracked)(gr)
	must.Nil(t, err)
	must.Length(t, 10, tracked)
	r, ok := tracked.Recv()
	must.True(t, ok)
	must.True(t, r)

	tracked = channel.NewUnbounded[bool]()
	workTracked = func(_ bool) error { tracked.Send(true); return nil }
	err = concurrent.GoEachRoutine(items, workTracked)(gr)
	must.Nil(t, err)
	must.Length(t, 10, tracked)
	r, ok = tracked.Recv()
	must.True(t, ok)
	must.True(t, r)
}

func TestGroup(t *testing.T) {
	ctx := context.Background()
	var err []error
	workNone := func() error { return nil }
	group, _ := concurrent.NewGroupContext(ctx)
	group.Go(workNone)
	err = group.Wait()
	must.Nil(t, err)
	group.Go(workNone)
	err = group.Wait()
	must.Nil(t, err)

	tracked := channel.NewUnbounded[bool]()
	workTracked := func() error { tracked.Send(true); return nil }
	err = group.Wait()
	must.Nil(t, err)
	must.Length(t, 0, tracked)

	tracked = channel.NewUnbounded[bool]()
	group.Go(workTracked)
	group.Go(workTracked)
	err = group.Wait()
	must.Nil(t, err)
	must.Length(t, 2, tracked)
	r, ok := tracked.Recv()
	must.True(t, ok)
	must.True(t, r)
	r, ok = tracked.Recv()
	must.True(t, ok)
	must.True(t, r)
}

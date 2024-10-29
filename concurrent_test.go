package concurrent_test

import (
	"errors"
	"testing"

	"github.com/gregwebs/go-concurrent"
	"github.com/stretchr/testify/assert"
)

func TestConcurrent(t *testing.T) {
	var err []error
	workNone := func(_ int) error { return nil }
	err = concurrent.GoN(0, workNone)
	assert.Nil(t, err)
	err = concurrent.GoN(2, workNone)
	assert.Nil(t, err)

	tracked := make([]bool, 10)
	workTracked := func(i int) error { tracked[i] = true; return nil }
	err = concurrent.GoN(0, workTracked)
	assert.Nil(t, err)
	assert.False(t, tracked[0])

	tracked = make([]bool, 10)
	err = concurrent.GoN(2, workTracked)
	assert.Nil(t, err)
	assert.False(t, tracked[2])
	assert.True(t, tracked[1])
	assert.True(t, tracked[0])
}

func TestChannelMerge(t *testing.T) {
	{
		c1 := make(chan error)
		c2 := make(chan error)
		close(c1)
		close(c2)
		err, ok := concurrent.TryRecv(concurrent.ChannelMerge(c1, c2))
		assert.False(t, ok)
		assert.Nil(t, err)
	}

	{
		c1 := make(chan error)
		c2 := make(chan error)
		go func() {
			c1 <- errors.New("c1")
			c2 <- errors.New("c2")
			close(c1)
			close(c2)
		}()
		merged := concurrent.ChannelMerge(c1, c2)
		_, ok := <-merged
		assert.True(t, ok)
		_, ok = <-merged
		assert.True(t, ok)
		_, ok = <-merged
		assert.False(t, ok)
	}
}

package channel_test

import (
	"errors"
	"testing"

	"github.com/gregwebs/go-concurrent"
	"github.com/gregwebs/go-concurrent/channel"
	"github.com/shoenig/test/must"
)

func TestChannelMerge(t *testing.T) {
	{
		c1 := make(chan error)
		c2 := make(chan error)
		close(c1)
		close(c2)
		err, ok := concurrent.TryRecv(channel.ChannelMerge(c1, c2))
		must.False(t, ok)
		must.Nil(t, err)
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
		merged := channel.ChannelMerge(c1, c2)
		_, ok := <-merged
		must.True(t, ok)
		_, ok = <-merged
		must.True(t, ok)
		_, ok = <-merged
		must.False(t, ok)
	}
}

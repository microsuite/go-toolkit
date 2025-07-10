package grpcpool

import (
	"testing"
	"time"
)

func TestTimerPool(t *testing.T) {
	pool := NewTimerPool()

	timer := pool.Get(5 * time.Second)
	defer pool.Put(timer)

	<-timer.C

	t.Logf("Timer expired\n")
}

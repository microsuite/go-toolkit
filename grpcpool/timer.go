package grpcpool

import (
	"sync"
	"time"
)

type TimerPool struct {
	pool sync.Pool
}

func NewTimerPool() *TimerPool {
	return &TimerPool{
		pool: sync.Pool{
			New: func() interface{} {
				return time.NewTimer(time.Hour)
			},
		},
	}
}

func (p *TimerPool) Get(d time.Duration) *time.Timer {
	timer := p.pool.Get().(*time.Timer)
	timer.Reset(d)
	return timer
}

func (p *TimerPool) Put(timer *time.Timer) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	p.pool.Put(timer)
}

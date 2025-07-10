package timingwheel

import (
	"sync"
	"time"
)

// truncate returns the result of rounding x toward zero to a multiple of m.
// If m <= 0, Truncate returns x unchanged.
func truncate(x, m int64) int64 {
	if m <= 0 {
		return x
	}
	return x - x%m
}

func timeToS(t time.Time) int64 {
	return t.UnixNano() / int64(time.Second)
}

func sToTime(t int64) time.Time {
	return time.Unix(0, t*int64(time.Second)).UTC()
}

type waitGroupWrapper struct {
	sync.WaitGroup
}

func (w *waitGroupWrapper) Wrap(cb func()) {
	w.Add(1)
	go func() {
		cb()
		w.Done()
	}()
}

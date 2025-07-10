package timingwheel

import (
	"log"
	"testing"
	"time"
)

func TestTimingWheel_AfterFunc(t *testing.T) {
	tw := NewTimingWheel(time.Second, 10)
	tw.Start()
	log.Printf("启动时间轮 \n")
	defer tw.Stop()

	durations := []time.Duration{
		5 * time.Second,
		2 * time.Second,
		1 * time.Minute,
	}

	for _, d := range durations {
		dTime := d
		tw.AfterFunc(d, func() {
			log.Printf("this is delay %v task \n", dTime)
		})

		if dTime == 5*time.Second {
			tw.AfterFunc(dTime, func() {
				log.Printf("this is delay %v task2 \n", dTime)
			})
		}
	}

	time.Sleep(time.Second * 100)
}

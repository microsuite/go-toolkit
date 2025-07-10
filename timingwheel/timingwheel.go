package timingwheel

import (
	"errors"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/microsuite/go-toolkit/container/dqueue"
)

type TimingWheel struct {
	// 基本时间跨度
	tickMs int64
	// 时间轮队列长度
	wheelSize int64
	// 总跨度
	interval int64
	// 表盘指针
	currentTime int64
	// 时间格队列
	buckets []*Bucket
	// 延时队列
	queue *dqueue.DelayQueue

	// 上层时间轮引用，可以通过Add函数去并发的读写, type: *TimingWheel
	overflowWheel unsafe.Pointer

	exitC     chan struct{}
	waitGroup waitGroupWrapper
}

func NewTimingWheel(tickMs time.Duration, wheelSize int64) *TimingWheel {
	tick := int64(tickMs / time.Second)
	if tickMs <= 0 {
		panic(errors.New("tickMs must be greater than or equal to 1s"))
	}

	startMs := timeToS(time.Now().UTC())
	return newTimingWheel(tick, wheelSize, startMs, dqueue.NewDelayQueue(int(wheelSize)))
}

func newTimingWheel(tickMs int64, wheelSize int64, startMs int64, queue *dqueue.DelayQueue) *TimingWheel {
	buckets := make([]*Bucket, wheelSize)
	for i := range buckets {
		buckets[i] = NewBucket()
	}

	return &TimingWheel{
		tickMs:      tickMs,
		wheelSize:   wheelSize,
		currentTime: truncate(startMs, tickMs),
		interval:    tickMs * wheelSize,
		buckets:     buckets,
		queue:       queue,
		exitC:       make(chan struct{}),
	}
}

// 增加延时任务至时间轮
func (tw *TimingWheel) add(t *TimerTaskEntity) bool {
	currentTime := atomic.LoadInt64(&tw.currentTime)
	if t.DelayTime < currentTime+tw.tickMs {
		return false
	} else if t.DelayTime < currentTime+tw.interval {
		// 写入到当前时间轮
		virtualID := t.DelayTime / tw.tickMs
		b := tw.buckets[virtualID%tw.wheelSize]
		b.Add(t)

		// 当前时间格写入到延时队列 ps. 延时队列是小根堆，时间格的过期时间是优先级参照，所以，越早到期的时间格越在队头
		if b.SetExpiration(t.DelayTime) {
			tw.queue.Add(b, b.Expiration())
		}
		return true
	} else {
		// 超出当前时间轮最大范畴，写入到上层时间轮
		overflowWheel := atomic.LoadPointer(&tw.overflowWheel)
		// 上层时间轮不存在 则创建新的时间轮。新时间轮的ticketMs = 当前时间轮的interval ；上层时间轮的开始时间startMs为当前时间轮表盘指针时间
		if overflowWheel == nil {
			atomic.CompareAndSwapPointer(&tw.overflowWheel, nil, unsafe.Pointer(newTimingWheel(tw.interval, tw.wheelSize, currentTime, tw.queue)))
			overflowWheel = atomic.LoadPointer(&tw.overflowWheel)
		}
		return (*TimingWheel)(overflowWheel).add(t)
	}
}

func (tw *TimingWheel) addOrRun(t *TimerTaskEntity) {
	if !tw.add(t) {
		// 如果无法添加，则表示该延时任务执行时间小于当前时间轮表盘指针指向的时间（换句话说，该延时任务已过期），则立即执行
		go t.Task()
	}
}

// advanceClock 推进时间轮时钟到指定的过期时间
func (tw *TimingWheel) advanceClock(expiration int64) {
	currentTime := atomic.LoadInt64(&tw.currentTime)
	if expiration >= currentTime+tw.tickMs {
		currentTime = truncate(expiration, tw.tickMs)
		atomic.StoreInt64(&tw.currentTime, currentTime)

		overflowWheel := atomic.LoadPointer(&tw.overflowWheel)
		if overflowWheel != nil {
			(*TimingWheel)(overflowWheel).advanceClock(currentTime)
		}
	}
}

// Start 时间轮启动
// 开启两个协程
//   - 协程1不断地从延时队列中取出最近一次到期的任务；
//   - 协程2根据获取到的任务的过期时间，去推进时间轮指针转动，以及去触发相应的延时任务（真实执行 or 任务降级）
func (tw *TimingWheel) Start() {
	tw.waitGroup.Wrap(func() {
		tw.queue.Poll(tw.exitC, func() int64 {
			return timeToS(time.Now().UTC())
		})
	})

	tw.waitGroup.Wrap(func() {
		for {
			select {
			case elem := <-tw.queue.Chan:
				b := elem.(*Bucket)
				tw.advanceClock(b.Expiration())
				b.Flush(tw.addOrRun)
			case <-tw.exitC:
				return
			}
		}
	})
}

func (tw *TimingWheel) Stop() {
	close(tw.exitC)
	tw.waitGroup.Wait()
}

func (tw *TimingWheel) AfterFunc(d time.Duration, fn func()) *TimerTaskEntity {
	t := &TimerTaskEntity{
		DelayTime: timeToS(time.Now().UTC().Add(d)),
		Task:      fn,
	}
	tw.addOrRun(t)
	return t
}

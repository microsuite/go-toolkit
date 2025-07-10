package timingwheel

import (
	"container/list"
	"sync"
	"sync/atomic"
)

// TimerTaskList 任务列表，双向链表。这里直接使用go内置的List对象
type TimerTaskList = list.List

// Bucket 时间格
type Bucket struct {
	expiration int64 // 时间格的到期时间，这个时间是时间格内存储定时任务的到期时间

	mu       sync.Mutex
	TaskList *TimerTaskList
}

func NewBucket() *Bucket {
	return &Bucket{
		TaskList:   list.New(),
		expiration: -1,
	}
}

func (b *Bucket) Expiration() int64 {
	return atomic.LoadInt64(&b.expiration)
}

func (b *Bucket) SetExpiration(expiration int64) bool {
	return atomic.SwapInt64(&b.expiration, expiration) != expiration
}

func (b *Bucket) Add(t *TimerTaskEntity) {
	b.mu.Lock()

	e := b.TaskList.PushBack(t)
	t.setBucket(b)
	t.element = e

	b.mu.Unlock()
}

func (b *Bucket) remove(t *TimerTaskEntity) bool {
	// 检查当前延时任务是否属于当前桶
	if t.getBucket() != b {
		return false
	}
	b.TaskList.Remove(t.element)
	t.setBucket(nil)
	t.element = nil
	return true
}

func (b *Bucket) Remove(t *TimerTaskEntity) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.remove(t)
}

// Flush 延时任务降级，重新插入到下层时间轮中
func (b *Bucket) Flush(reinsert func(*TimerTaskEntity)) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for e := b.TaskList.Front(); e != nil; {
		next := e.Next()
		t := e.Value.(*TimerTaskEntity)
		b.remove(t)
		reinsert(t)
		e = next
	}
	// 当前桶所有延时任务降级完成后，该桶过期时间重置为-1，该桶不再有效
	b.SetExpiration(-1)
}

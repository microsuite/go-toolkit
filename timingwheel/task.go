package timingwheel

import (
	"container/list"
	"sync/atomic"
	"unsafe"
)

// TimerTaskEntity 延时任务
type TimerTaskEntity struct {
	// 延时时间
	DelayTime int64
	// 任务执行函数
	Task func()

	// type: *bucket 保存当前延时任务所在的时间格，使用桶指针，可通过原子操作并发更新/读取
	b unsafe.Pointer

	// 延时任务所在的双向链表中的节点元素
	element *list.Element
}

func (t *TimerTaskEntity) getBucket() *Bucket {
	return (*Bucket)(atomic.LoadPointer(&t.b))
}

func (t *TimerTaskEntity) setBucket(b *Bucket) {
	atomic.StorePointer(&t.b, unsafe.Pointer(b))
}

// Stop 停止延时任务的执行
func (t *TimerTaskEntity) Stop() bool {
	stopped := false
	for b := t.getBucket(); b != nil; b = t.getBucket() {
		// 如果时间格尚未过期/执行，则从时间格中删除这个延时任务
		stopped = b.Remove(t)
	}
	return stopped
}

package queue

import (
	"container/heap"
	"strconv"
	"sync"
	"zee-mirror/internal/metrics"
)

type PriorityTask struct {
	Task     any
	Priority int
	Index    int
}

type PriorityQueue struct {
	cond  *sync.Cond
	items []*PriorityTask
	mu    sync.Mutex
}

func NewPriorityQueue() *PriorityQueue {
	pq := &PriorityQueue{
		items: make([]*PriorityTask, 0),
	}
	pq.cond = sync.NewCond(&pq.mu)
	return pq
}

func (pq *PriorityQueue) Len() int {
	return len(pq.items)
}

func (pq *PriorityQueue) Less(i, j int) bool {
	return pq.items[i].Priority > pq.items[j].Priority
}

func (pq *PriorityQueue) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
	pq.items[i].Index = i
	pq.items[j].Index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(pq.items)
	item := x.(*PriorityTask)
	item.Index = n
	pq.items = append(pq.items, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := pq.items
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.Index = -1
	pq.items = old[0 : n-1]
	return item
}

func (pq *PriorityQueue) Enqueue(task any, priority int) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	item := &PriorityTask{
		Task:     task,
		Priority: priority,
	}
	heap.Push(pq, item)
	metrics.QueueDepth.WithLabelValues(strconv.Itoa(priority)).Inc()
	pq.cond.Signal()
}

func (pq *PriorityQueue) Dequeue() any {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	for len(pq.items) == 0 {
		pq.cond.Wait()
	}

	item := heap.Pop(pq).(*PriorityTask)
	metrics.QueueDepth.WithLabelValues(strconv.Itoa(item.Priority)).Dec()
	return item.Task
}

func (pq *PriorityQueue) DequeueNonBlocking() any {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if len(pq.items) == 0 {
		return nil
	}

	item := heap.Pop(pq).(*PriorityTask)
	metrics.QueueDepth.WithLabelValues(strconv.Itoa(item.Priority)).Dec()
	return item.Task
}

func (pq *PriorityQueue) Count() int {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	return len(pq.items)
}

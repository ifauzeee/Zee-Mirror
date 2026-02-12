package queue

import (
	"testing"
)

type MockTask struct {
	ID string
}

func TestPriorityQueue(t *testing.T) {
	pq := NewPriorityQueue()

	task1 := &MockTask{ID: "low_priority"}
	task2 := &MockTask{ID: "high_priority"}
	task3 := &MockTask{ID: "medium_priority"}

	pq.Enqueue(task1, 0)
	pq.Enqueue(task2, 100)
	pq.Enqueue(task3, 50)

	if pq.Count() != 3 {
		t.Errorf("expected count 3, got %d", pq.Count())
	}

	item1 := pq.DequeueNonBlocking()
	if item1 != task2 {
		t.Errorf("expected task2 (high), got %v", item1)
	}

	item2 := pq.DequeueNonBlocking()
	if item2 != task3 {
		t.Errorf("expected task3 (medium), got %v", item2)
	}

	item3 := pq.DequeueNonBlocking()
	if item3 != task1 {
		t.Errorf("expected task1 (low), got %v", item3)
	}

	item4 := pq.DequeueNonBlocking()
	if item4 != nil {
		t.Errorf("expected nil, got %v", item4)
	}
}

func TestUserRateLimiter(t *testing.T) {
	limiter := NewUserRateLimiter(60, 1)

	userID := int64(12345)

	if !limiter.Allow(userID) {
		t.Errorf("expected first request allowed")
	}

	if limiter.Allow(userID) {
		t.Errorf("expected second request blocked (burst 1)")
	}

	if !limiter.Allow(67890) {
		t.Errorf("expected different user allowed")
	}
}

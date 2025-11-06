package core

import (
	"sync"

	"github.com/govdbot/govd/internal/logger"
)

type TaskQueue struct {
	mu      sync.Mutex
	tasks   map[string]bool
	waiting map[string][]chan struct{}
}

var taskQueue = &TaskQueue{
	tasks:   make(map[string]bool),
	waiting: make(map[string][]chan struct{}),
}

// attempts to acquire the lock for processing a task with the given key.
func (tq *TaskQueue) Acquire(key string) {
	tq.mu.Lock()

	if tq.tasks[key] {
		// task is being processed, create a channel to wait on
		logger.L.Debugf("waiting for existing task with key: %s", key)

		done := make(chan struct{})
		tq.waiting[key] = append(tq.waiting[key], done)
		tq.mu.Unlock()

		<-done
		return
	}
	tq.tasks[key] = true
	tq.mu.Unlock()
}

// releases the lock for the given key and notifies all waiting tasks.
func (tq *TaskQueue) Release(key string) {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	delete(tq.tasks, key)

	if waiters, exists := tq.waiting[key]; exists {
		for _, done := range waiters {
			close(done)
		}
		delete(tq.waiting, key)
	}
}

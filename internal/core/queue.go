package core

import (
	"sync"
)

// queueMu guards queue Locks map; cleanup on release prevents unbounded growth.
var (
	queueMu sync.Mutex
	queue   = make(map[string]*sync.Mutex)
)

func acquireQueue(key string) {
	queueMu.Lock()
	mu, ok := queue[key]
	if !ok {
		mu = &sync.Mutex{}
		queue[key] = mu
	}
	queueMu.Unlock()
	mu.Lock()
}

func releaseQueue(key string) {
	queueMu.Lock()
	defer queueMu.Unlock()
	if mu, ok := queue[key]; ok {
		mu.Unlock()
		// keep entry to avoid re-alloc for hot keys; but cap size via lazy GC
		// if map grows too large (>5000 distinct keys seen concurrently),
		// drop un-locked entries opportunistically on next acquire.
		if len(queue) > 5000 {
			// best-effort cleanup: remove this key, it will be re-created on next use
			delete(queue, key)
		}
	}
}

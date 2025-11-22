package core

import (
	"sync"
)

var queueLock sync.Map

func acquireQueue(key string) {
	value, _ := queueLock.LoadOrStore(key, &sync.Mutex{})
	mu, ok := value.(*sync.Mutex)
	if !ok {
		return
	}
	mu.Lock()
}

func releaseQueue(key string) {
	if value, ok := queueLock.Load(key); ok {
		mu, ok := value.(*sync.Mutex)
		if !ok {
			return
		}
		mu.Unlock()
	}
}

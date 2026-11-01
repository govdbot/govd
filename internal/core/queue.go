package core

import (
	"context"
	"sync"
)

type keyLock struct {
	token chan struct{}
	refs  int
}

var queueMu sync.Mutex
var queueKeys = map[string]*keyLock{}

func acquireQueue(ctx context.Context, key string) (func(), error) {
	queueMu.Lock()
	lock := queueKeys[key]
	if lock == nil {
		lock = &keyLock{token: make(chan struct{}, 1)}
		lock.token <- struct{}{}
		queueKeys[key] = lock
	}
	lock.refs++
	queueMu.Unlock()
	drop := func() {
		queueMu.Lock()
		defer queueMu.Unlock()
		lock.refs--
		if lock.refs == 0 {
			delete(queueKeys, key)
		}
	}
	select {
	case <-ctx.Done():
		drop()
		return nil, ctx.Err()
	case <-lock.token:
	}
	var once sync.Once
	return func() { once.Do(func() { lock.token <- struct{}{}; drop() }) }, nil
}

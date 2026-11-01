package core

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestQueueWaitCanBeCancelledAndKeysAreReleased(t *testing.T) {
	release, err := acquireQueue(context.Background(), "same")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := acquireQueue(ctx, "same"); err == nil {
		t.Fatal("cancelled wait succeeded")
	}
	release()
	for i := range 100 {
		r, e := acquireQueue(context.Background(), fmt.Sprint(i))
		if e != nil {
			t.Fatal(e)
		}
		r()
		r()
	}
	queueMu.Lock()
	n := len(queueKeys)
	queueMu.Unlock()
	if n != 0 {
		t.Fatalf("retained keys %d", n)
	}
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()
	r, e := acquireQueue(ctx2, "same")
	if e != nil {
		t.Fatal(e)
	}
	r()
}

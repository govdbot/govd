package admission

import (
	"sync"
	"testing"
	"time"
)

func TestGroupLimitsAndRecovery(t *testing.T) {
	g := New()
	now := time.Now()
	g.now = func() time.Time { return now }
	release, ok := g.Begin(101, -10, false)
	if !ok {
		t.Fatal("first rejected")
	}
	if _, ok := g.Begin(101, -10, false); ok {
		t.Fatal("same user active twice")
	}
	other, ok := g.Begin(202, -10, false)
	if !ok {
		t.Fatal("second rejected")
	}
	if _, ok := g.Begin(303, -20, false); ok {
		t.Fatal("global cap exceeded")
	}
	release()
	release()
	other()
	for range 2 {
		r, ok := g.Begin(101, -10, false)
		if !ok {
			t.Fatal("within rate rejected")
		}
		r()
	}
	if _, ok := g.Begin(101, -10, false); ok {
		t.Fatal("user rate exceeded")
	}
	now = now.Add(time.Minute + time.Second)
	r, ok := g.Begin(101, -10, false)
	if !ok {
		t.Fatal("rate did not recover")
	}
	r()
}
func TestGuardConcurrentClaims(t *testing.T) {
	g := New()
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := int64(1); i <= 20; i++ {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			<-start
			if r, ok := g.Begin(id, -1, false); ok {
				r()
				r()
			}
		}(i)
	}
	close(start)
	wg.Wait()
	if g.active != 0 {
		t.Fatal("active count leaked")
	}
}
func TestLightGateAndNotice(t *testing.T) {
	g := New()
	now := time.Now()
	g.now = func() time.Time { return now }
	for range 30 {
		if !g.AllowLight(101) {
			t.Fatal("early reject")
		}
	}
	if g.AllowLight(101) {
		t.Fatal("light gate unbounded")
	}
	if !g.Notify(101) || g.Notify(101) {
		t.Fatal("notice flood")
	}
	now = now.Add(11 * time.Minute)
	if !g.AllowLight(202) {
		t.Fatal("no recovery")
	}
	if _, old := g.users[101]; old {
		t.Fatal("idle actor retained")
	}
}

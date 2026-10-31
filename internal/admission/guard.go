package admission

import (
	"sync"
	"time"
)

type bucket struct {
	heavy, light []time.Time
	active       int
	seen, notice time.Time
}
type Guard struct {
	mu           sync.Mutex
	now          func() time.Time
	users, chats map[int64]*bucket
	heavy, light []time.Time
	active       int
	lastGC       time.Time
}

func New() *Guard {
	return &Guard{now: time.Now, users: map[int64]*bucket{}, chats: map[int64]*bucket{}}
}

var Default = New()

func recent(x []time.Time, now time.Time) []time.Time {
	i := 0
	for i < len(x) && !x[i].After(now.Add(-time.Minute)) {
		i++
	}
	return x[i:]
}
func (g *Guard) collect(now time.Time) {
	if now.Sub(g.lastGC) < time.Minute {
		return
	}
	g.lastGC = now
	for _, m := range []map[int64]*bucket{g.users, g.chats} {
		for k, b := range m {
			if b.active == 0 && now.Sub(b.seen) > 10*time.Minute {
				delete(m, k)
			}
		}
	}
}
func (g *Guard) get(m map[int64]*bucket, id int64, now time.Time) *bucket {
	if b := m[id]; b != nil {
		b.seen = now
		return b
	}
	if len(m) >= 4096 {
		return nil
	}
	b := &bucket{seen: now}
	m[id] = b
	return b
}

// Light admission happens before settings/database work, including inline previews.
func (g *Guard) AllowLight(actor int64) bool {
	if actor <= 0 {
		return false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	now := g.now()
	g.collect(now)
	u := g.get(g.users, actor, now)
	if u == nil {
		return false
	}
	u.light = recent(u.light, now)
	g.light = recent(g.light, now)
	if len(u.light) >= 30 || len(g.light) >= 180 {
		return false
	}
	u.light = append(u.light, now)
	g.light = append(g.light, now)
	return true
}

// Begin bounds both work rate and active work. No unbounded waiting goroutines.
func (g *Guard) Begin(actor, chat int64, trustedPrivate bool) (func(), bool) {
	if actor <= 0 || chat == 0 {
		return nil, false
	}
	g.mu.Lock()
	now := g.now()
	g.collect(now)
	u := g.get(g.users, actor, now)
	c := g.get(g.chats, chat, now)
	if u == nil || c == nil {
		g.mu.Unlock()
		return nil, false
	}
	u.heavy = recent(u.heavy, now)
	c.heavy = recent(c.heavy, now)
	g.heavy = recent(g.heavy, now)
	userLimit := 3
	if trustedPrivate {
		userLimit = 6
	}
	if u.active >= 1 || g.active >= 2 || len(u.heavy) >= userLimit || len(c.heavy) >= 6 || len(g.heavy) >= 12 {
		g.mu.Unlock()
		return nil, false
	}
	u.active++
	c.active++
	g.active++
	u.heavy = append(u.heavy, now)
	c.heavy = append(c.heavy, now)
	g.heavy = append(g.heavy, now)
	g.mu.Unlock()
	var once sync.Once
	return func() { once.Do(func() { g.mu.Lock(); defer g.mu.Unlock(); u.active--; c.active--; g.active-- }) }, true
}
func (g *Guard) Notify(actor int64) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := g.now()
	u := g.get(g.users, actor, now)
	if u == nil || now.Sub(u.notice) < 15*time.Second {
		return false
	}
	u.notice = now
	return true
}

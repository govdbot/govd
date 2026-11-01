package core

import (
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/models"
	"sync"
	"sync/atomic"
	"testing"
)

func TestSecurityTaskOwnerAndSingleUse(t *testing.T) {
	id := "security-owner-fixture"
	defer RemoveTask(id)
	task := &models.ExtractorContext{Chat: &database.GetOrCreateChatRow{ChatID: 101}}
	if !AddTask(id, task) {
		t.Fatal("initial add failed")
	}
	if got, ok := ClaimTask(id, 202); ok || got != nil {
		t.Fatal("foreign actor claimed task")
	}
	if _, ok := GetTask(id); !ok {
		t.Fatal("foreign claim consumed owner's task")
	}
	var successes atomic.Int32
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if got, ok := ClaimTask(id, 101); ok {
				if got != task {
					t.Error("wrong task")
				}
				successes.Add(1)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatalf("claims=%d", successes.Load())
	}
	if _, ok := ClaimTask(id, 101); ok {
		t.Fatal("replayed task accepted")
	}
}

func TestSecurityTaskCollisionDoesNotReplaceOwner(t *testing.T) {
	id := "security-collision-fixture"
	defer RemoveTask(id)
	first := &models.ExtractorContext{Chat: &database.GetOrCreateChatRow{ChatID: 101}}
	AddTask(id, first)
	if AddTask(id, &models.ExtractorContext{Chat: &database.GetOrCreateChatRow{ChatID: 202}}) {
		t.Fatal("duplicate accepted")
	}
	if got, ok := ClaimTask(id, 101); !ok || got != first {
		t.Fatal("original task replaced")
	}
}

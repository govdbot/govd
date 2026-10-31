package handlers

import (
	"context"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/core"
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/models"
	"go.uber.org/zap"
	"testing"
)

func TestSecurityForeignInlineDoesNotRunOrCancelOwner(t *testing.T) {
	old := logger.L
	logger.L = zap.NewNop().Sugar()
	defer func() { logger.L = old }()
	called := false
	cancelled := false
	id := "security-handler-fixture"
	defer core.RemoveTask(id)
	task := &models.ExtractorContext{ContentID: "fixture", Chat: &database.GetOrCreateChatRow{ChatID: 101, Language: "en"}, Context: context.Background(), CancelFunc: func() { cancelled = true }, FilesTracker: models.NewFilesTracker(), Extractor: &models.Extractor{ID: "fixture", GetFunc: func(*models.ExtractorContext) (*models.ExtractorResponse, error) {
		called = true
		return nil, core.ErrNoMedia
	}}}
	core.AddTask(id, task)
	actor := gotgbot.User{Id: 202}
	ctx := &ext.Context{Update: &gotgbot.Update{ChosenInlineResult: &gotgbot.ChosenInlineResult{ResultId: id, From: actor}}, EffectiveUser: &actor}
	InlineResultHandler(nil, ctx)
	if called || cancelled {
		t.Fatal("foreign actor executed or cancelled owner's task")
	}
	if got, ok := core.GetTask(id); !ok || got != task {
		t.Fatal("foreign actor removed task")
	}
}

func TestSecurityStatsCallbackRequiresAdminBeforeDatabase(t *testing.T) {
	old := config.Env.Admins
	config.Env.Admins = []int64{101}
	defer func() { config.Env.Admins = old }()
	actor := gotgbot.User{Id: 202}
	ctx := &ext.Context{Update: &gotgbot.Update{CallbackQuery: &gotgbot.CallbackQuery{From: actor, Data: "stats:all"}}, EffectiveUser: &actor, EffectiveChat: &gotgbot.Chat{Id: 202, Type: "private"}}
	if err := StatsCallbackHandler(nil, ctx); err != ext.EndGroups {
		t.Fatalf("unexpected result: %v", err)
	}
}

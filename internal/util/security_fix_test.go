package util

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/govdbot/govd/internal/config"
	"testing"
)

func TestSecurityAdminCallbackUsesActorNotBotMessageAuthor(t *testing.T) {
	old := config.Env.Admins
	config.Env.Admins = []int64{101}
	defer func() { config.Env.Admins = old }()
	actor := gotgbot.User{Id: 101}
	ctx := &ext.Context{Update: &gotgbot.Update{}, EffectiveUser: &actor, EffectiveChat: &gotgbot.Chat{Id: 101, Type: "private"}, EffectiveMessage: &gotgbot.Message{From: &gotgbot.User{Id: 999, IsBot: true}}}
	if !IsBotAdmin(ctx) {
		t.Fatal("admin callback actor rejected")
	}
	actor.Id = 202
	if IsBotAdmin(ctx) {
		t.Fatal("non-admin callback actor accepted")
	}
	if IsBotAdmin(nil) || IsBotAdmin(&ext.Context{}) {
		t.Fatal("incomplete identity accepted")
	}
}

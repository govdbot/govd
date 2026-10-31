package handlers

import (
	"slices"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"
	"github.com/govdbot/govd/internal/admission"
	"github.com/govdbot/govd/internal/core"
	"github.com/govdbot/govd/internal/extractors"
	"github.com/govdbot/govd/internal/localization"
	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/util"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func URLFilter(msg *gotgbot.Message) bool {
	return msg != nil && !message.Command(msg) && util.URLFromMessage(msg) != ""
}

func URLHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	for _, raw := range util.URLsFromMessage(ctx.EffectiveMessage) {
		if extractors.InputSupported(raw) {
			handleURL(bot, ctx, raw)
		}
	}
	return ext.EndGroups
}

func handleURL(bot *gotgbot.Bot, ctx *ext.Context, url string) error {
	message := ctx.EffectiveMessage

	if url == "" {
		return ext.EndGroups
	}

	if util.HasHashtagEntity(message, "skip") {
		return ext.EndGroups
	}
	if ctx.EffectiveUser == nil || ctx.EffectiveChat == nil {
		return ext.EndGroups
	}
	release, allowed := admission.Default.Begin(ctx.EffectiveUser.Id, ctx.EffectiveChat.Id, util.IsBotAdmin(ctx))
	if !allowed {
		if admission.Default.Notify(ctx.EffectiveUser.Id) {
			message.Reply(bot, localization.New(ctx.EffectiveUser.LanguageCode).T(&i18n.LocalizeConfig{MessageID: "ErrorBusy"}), nil)
		}
		return ext.EndGroups
	}
	defer release()

	extractorCtx := extractors.FromURL(url)
	if extractorCtx == nil || extractorCtx.Extractor == nil {
		return ext.EndGroups
	}

	defer extractorCtx.CancelFunc()

	chat, err := util.ChatFromContext(ctx)
	if err != nil {
		logger.L.Errorf("failed to get settings from context: %v", err)
		return ext.EndGroups
	}
	if chat != nil && slices.Contains(chat.DisabledExtractors, extractorCtx.Extractor.ID) {
		return ext.EndGroups
	}
	extractorCtx.SetChat(chat)

	err = util.SendTypingAction(bot, chat.ChatID)
	if err != nil {
		core.HandleError(bot, ctx, extractorCtx, err)
		return ext.EndGroups
	}

	err = core.HandleDownloadTask(bot, ctx, extractorCtx)
	if err != nil {
		core.HandleError(bot, ctx, extractorCtx, err)
		return ext.EndGroups
	}

	return ext.EndGroups
}

package handlers

import (
	"slices"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"
	"github.com/govdbot/govd/internal/core"
	"github.com/govdbot/govd/internal/extractors"
	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/util"
)

func URLFilter(msg *gotgbot.Message) bool {
	return message.Text(msg) &&
		!message.Command(msg) &&
		message.Entity("url")(msg)
}

func URLHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	message := ctx.EffectiveMessage

	url := util.URLFromMessage(message)
	if url == "" {
		return ext.EndGroups
	}

	if util.HasHashtagEntity(message, "skip") {
		return ext.EndGroups
	}

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

	extractorCtx.User = message.From
	extractorCtx.Comment = util.ExtractCommentFromMessage(message)

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

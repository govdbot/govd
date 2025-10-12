package handlers

import (
	"context"
	"time"

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
	chat := message.Chat

	url := util.URLFromMessage(message)
	if url == "" {
		return ext.EndGroups
	}

	if util.HasHashtagEntity(message, "skip") {
		return ext.EndGroups
	}

	taskCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	extractorCtx := extractors.FromURL(taskCtx, url)
	if extractorCtx == nil || extractorCtx.Extractor == nil {
		return ext.EndGroups
	}

	settings, err := util.SettingsFromContext(ctx)
	if err != nil {
		logger.L.Errorf("failed to get settings from context: %v", err)
		return ext.EndGroups
	}
	extractorCtx.SetSettings(settings)

	util.SendTypingAction(bot, chat.Id)

	err = core.HandleDownloadTask(bot, ctx, extractorCtx)
	if err != nil {
		core.HandleError(bot, ctx, extractorCtx, err)
	}

	return nil
}

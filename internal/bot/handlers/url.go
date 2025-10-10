package handlers

import (
	"context"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"
	"github.com/govdbot/govd/internal/core"
	"github.com/govdbot/govd/internal/extractors"
	"github.com/govdbot/govd/internal/util"
)

func URLFilter(msg *gotgbot.Message) bool {
	return message.Text(msg) &&
		!message.Command(msg) &&
		message.Entity("url")(msg)
}

func URLHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	url := util.URLFromMessage(ctx.EffectiveMessage)
	if url == "" {
		return ext.EndGroups
	}

	// TODO: skip if hashtag

	taskCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	extractorCtx := extractors.FromURL(taskCtx, url)
	if extractorCtx == nil || extractorCtx.Extractor == nil {
		return ext.EndGroups
	}

	chat := ctx.EffectiveMessage.Chat
	settings, err := util.SettingsFromContext(ctx)
	extractorCtx.SetSettings(settings)

	util.SendTypingAction(bot, chat.Id)

	err = core.HandleDownloadTask(bot, ctx, extractorCtx)
	if err != nil {
		return err
	}

	return nil
}

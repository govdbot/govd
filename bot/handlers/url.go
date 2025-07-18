package handlers

import (
	"context"
	"time"

	"github.com/govdbot/govd/bot/core"
	"github.com/govdbot/govd/database"
	extractors "github.com/govdbot/govd/ext"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"
)

func URLHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	messageURL := getMessageURL(ctx.EffectiveMessage)
	if messageURL == "" {
		return nil
	}
	if shouldSkip(ctx.EffectiveMessage) {
		// skip processing if the message contains #skip hashtag
		return nil
	}
	dlCtx, err := extractors.CtxByURL(messageURL)
	if err != nil {
		return nil
	}
	if dlCtx == nil || dlCtx.Extractor == nil {
		return nil
	}
	dlCtx.IsSpoiler = isSpoiler(ctx.EffectiveMessage)
	userID := ctx.EffectiveMessage.From.Id
	if ctx.EffectiveMessage.Chat.Type != gotgbot.ChatTypePrivate {
		settings, err := database.GetGroupSettings(ctx.EffectiveMessage.Chat.Id)
		if err != nil {
			return err
		}
		dlCtx.GroupSettings = settings
	}
	if userID != 1087968824 {
		// groupAnonymousBot
		_, err = database.GetUser(userID)
		if err != nil {
			return err
		}
	}

	taskCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	err = core.HandleDownloadRequest(
		bot, ctx, taskCtx, dlCtx)
	if err != nil {
		if dlCtx.GroupSettings != nil && *dlCtx.GroupSettings.Silent {
			return nil
		}
		core.HandleErrorMessage(bot, ctx, err)
	}
	return nil
}

func URLFilter(msg *gotgbot.Message) bool {
	return message.Text(msg) &&
		!message.Command(msg) &&
		message.Entity("url")(msg)
}

func hashtagEntity(msg *gotgbot.Message, entity string) bool {
	entity = "#" + entity
	for _, ent := range msg.Entities {
		if ent.Type != "hashtag" {
			continue
		}
		parsedEntity := gotgbot.ParseEntity(
			msg.Text,
			ent,
		)
		if parsedEntity.Text == entity {
			return true
		}
	}
	return false
}

func shouldSkip(msg *gotgbot.Message) bool {
	return hashtagEntity(msg, "skip")
}

func isSpoiler(msg *gotgbot.Message) bool {
	return hashtagEntity(msg, "spoiler") ||
		hashtagEntity(msg, "nsfw")
}

func getMessageURL(msg *gotgbot.Message) string {
	for _, entity := range msg.Entities {
		if entity.Type != "url" {
			continue
		}
		parsedEntity := gotgbot.ParseEntity(
			msg.Text,
			entity,
		)
		return parsedEntity.Text
	}
	return ""
}

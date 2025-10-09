package handlers

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"
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

	extractorCtx := extractors.FromURL(url)
	if extractorCtx == nil || extractorCtx.Extractor == nil {
		return ext.EndGroups
	}

	return nil
}

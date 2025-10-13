package handlers

import (
	"context"
	"slices"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/database"
)

func DecodeErrorHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	chatType := ctx.EffectiveChat.Type
	if chatType != gotgbot.ChatTypePrivate {
		return ext.EndGroups
	}
	userID := ctx.EffectiveMessage.From.Id

	if !slices.Contains(config.Env.Admins, userID) {
		return ext.EndGroups
	}

	args := ctx.Args()
	if len(args) < 2 {
		return ext.EndGroups
	}
	errorID := args[1]

	errorMessage, err := database.Q().GetErrorByID(
		context.Background(),
		errorID,
	)
	if err != nil {
		return ext.EndGroups
	}

	ctx.EffectiveMessage.Reply(bot, errorMessage, nil)
	return ext.EndGroups
}

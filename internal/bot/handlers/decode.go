package handlers

import (
	"context"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/util"
)

func DecodeErrorHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	ok := util.IsBotAdmin(ctx)
	if !ok {
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

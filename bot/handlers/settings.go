package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/govdbot/govd/database"
	"github.com/govdbot/govd/util"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

func SettingsHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.EffectiveMessage.Chat.Type == gotgbot.ChatTypePrivate {
		ctx.EffectiveMessage.Reply(
			bot,
			"use this command in group chats only",
			nil,
		)
		return nil
	}
	settings, err := database.GetGroupSettings(ctx.EffectiveMessage.Chat.Id)
	if err != nil {
		return err
	}
	ctx.EffectiveMessage.Reply(
		bot,
		fmt.Sprintf(
			"settings for this group\n\n"+
				"captions: %s\n"+
				"nsfw: %s\n"+
				"silent mode: %s\n"+
				"media group limit: %d\n",
			strconv.FormatBool(*settings.Captions),
			strconv.FormatBool(*settings.NSFW),
			strconv.FormatBool(*settings.Silent),
			settings.MediaGroupLimit,
		),
		nil,
	)
	return nil
}

func CaptionsHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.EffectiveMessage.Chat.Type == gotgbot.ChatTypePrivate {
		return nil
	}

	chatID := ctx.EffectiveMessage.Chat.Id
	userID := ctx.EffectiveMessage.From.Id

	args := ctx.Args()
	if len(args) != 2 {
		ctx.EffectiveMessage.Reply(
			bot,
			"usage: /captions (true|false)",
			nil,
		)
		return nil
	}
	if !util.IsUserAdmin(bot, chatID, userID) {
		ctx.EffectiveMessage.Reply(
			bot,
			"you don't have permission to change settings",
			nil,
		)
		return nil
	}
	userInput := strings.ToLower(args[1])
	value, err := strconv.ParseBool(userInput)
	if err != nil {
		ctx.EffectiveMessage.Reply(
			bot,
			fmt.Sprintf("invalid value (%s), use true or false", userInput),
			nil,
		)
		return err
	}
	settings, err := database.GetGroupSettings(chatID)
	if err != nil {
		return err
	}
	settings.Captions = &value
	err = database.UpdateGroupSettings(chatID, settings)
	if err != nil {
		return err
	}
	var message string
	if value {
		message = "captions enabled"
	} else {
		message = "captions disabled"
	}
	ctx.EffectiveMessage.Reply(
		bot,
		message,
		nil,
	)
	return nil
}

func NSFWHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.EffectiveMessage.Chat.Type == gotgbot.ChatTypePrivate {
		return nil
	}

	chatID := ctx.EffectiveMessage.Chat.Id
	userID := ctx.EffectiveMessage.From.Id

	args := ctx.Args()
	if len(args) != 2 {
		ctx.EffectiveMessage.Reply(
			bot,
			"usage: /nsfw (true|false)",
			nil,
		)
		return nil
	}
	if !util.IsUserAdmin(bot, chatID, userID) {
		ctx.EffectiveMessage.Reply(
			bot,
			"you don't have permission to change settings",
			nil,
		)
		return nil
	}
	userInput := strings.ToLower(args[1])
	value, err := strconv.ParseBool(userInput)
	if err != nil {
		ctx.EffectiveMessage.Reply(
			bot,
			fmt.Sprintf("invalid value (%s), use true or false", userInput),
			nil,
		)
		return err
	}
	settings, err := database.GetGroupSettings(chatID)
	if err != nil {
		return err
	}
	settings.NSFW = &value
	err = database.UpdateGroupSettings(chatID, settings)
	if err != nil {
		return err
	}
	var message string
	if value {
		message = "nsfw enabled"
	} else {
		message = "nsfw disabled"
	}
	ctx.EffectiveMessage.Reply(
		bot,
		message,
		nil,
	)
	return nil
}

func MediaGroupLimitHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.EffectiveMessage.Chat.Type == gotgbot.ChatTypePrivate {
		return nil
	}

	chatID := ctx.EffectiveMessage.Chat.Id
	userID := ctx.EffectiveMessage.From.Id

	args := ctx.Args()
	if len(args) != 2 {
		ctx.EffectiveMessage.Reply(
			bot,
			"usage: /limit (int)",
			nil,
		)
		return nil
	}
	if !util.IsUserAdmin(bot, chatID, userID) {
		ctx.EffectiveMessage.Reply(
			bot,
			"you don't have permission to change settings",
			nil,
		)
		return nil
	}
	value, err := strconv.Atoi(args[1])
	if err != nil {
		ctx.EffectiveMessage.Reply(
			bot,
			fmt.Sprintf("invalid value (%s), use a number", args[1]),
			nil,
		)
		return err
	}
	if value < 1 || value > 20 {
		ctx.EffectiveMessage.Reply(
			bot,
			"media group limit must be between 1 and 20",
			nil,
		)
		return nil
	}
	settings, err := database.GetGroupSettings(chatID)
	if err != nil {
		return err
	}
	settings.MediaGroupLimit = value
	err = database.UpdateGroupSettings(chatID, settings)
	if err != nil {
		return err
	}
	ctx.EffectiveMessage.Reply(
		bot,
		fmt.Sprintf("media group limit set to %d", value),
		nil,
	)
	return nil
}

func SilentHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.EffectiveMessage.Chat.Type == gotgbot.ChatTypePrivate {
		return nil
	}

	chatID := ctx.EffectiveMessage.Chat.Id
	userID := ctx.EffectiveMessage.From.Id

	args := ctx.Args()
	if len(args) != 2 {
		ctx.EffectiveMessage.Reply(
			bot,
			"usage: /silent (true|false)",
			nil,
		)
		return nil
	}
	if !util.IsUserAdmin(bot, chatID, userID) {
		ctx.EffectiveMessage.Reply(
			bot,
			"you don't have permission to change settings",
			nil,
		)
		return nil
	}
	userInput := strings.ToLower(args[1])
	value, err := strconv.ParseBool(userInput)
	if err != nil {
		ctx.EffectiveMessage.Reply(
			bot,
			fmt.Sprintf("invalid value (%s), use true or false", userInput),
			nil,
		)
		return err
	}
	settings, err := database.GetGroupSettings(chatID)
	if err != nil {
		return err
	}
	settings.Silent = &value
	err = database.UpdateGroupSettings(chatID, settings)
	if err != nil {
		return err
	}
	var message string
	if value {
		message = "silent mode enabled"
	} else {
		message = "silent mode disabled"
	}
	ctx.EffectiveMessage.Reply(
		bot,
		message,
		nil,
	)
	return nil
}

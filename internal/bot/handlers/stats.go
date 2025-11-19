package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/util"
	"github.com/jackc/pgx/v5/pgtype"
)

func StatsHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	ok := util.IsBotAdmin(ctx)
	if !ok {
		return ext.EndGroups
	}
	text, err := formatMessage("all")
	if err != nil {
		return err
	}
	ctx.EffectiveMessage.Reply(
		bot, text, &gotgbot.SendMessageOpts{
			ReplyMarkup: getStatsKeyboard(),
		},
	)
	return ext.EndGroups
}

func StatsCallbackHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	data := ctx.CallbackQuery.Data
	parts := strings.Split(data, ":")
	if len(parts) < 2 {
		return nil
	}
	period := parts[1]

	text, err := formatMessage(period)
	if err != nil {
		return err
	}
	ctx.CallbackQuery.Answer(bot, nil)
	ctx.EffectiveMessage.EditText(
		bot, text,
		&gotgbot.EditMessageTextOpts{
			ReplyMarkup: getStatsKeyboard(),
		},
	)
	return nil
}

func formatMessage(period string) (string, error) {
	var sinceDate time.Time
	var periodText string

	switch period {
	case "1d":
		sinceDate = time.Now().Add(-24 * time.Hour)
		periodText = "24 hours"
	case "7d":
		sinceDate = time.Now().Add(-7 * 24 * time.Hour)
		periodText = "7 days"
	case "30d":
		sinceDate = time.Now().Add(-30 * 24 * time.Hour)
		periodText = "30 days"
	default:
		sinceDate = time.Now().Add(-100 * 365 * 24 * time.Hour)
		periodText = "all time"
	}

	stats, err := database.Q().GetStats(
		context.Background(),
		pgtype.Timestamptz{
			Time:  sinceDate,
			Valid: true,
		},
	)
	if err != nil {
		return "", err
	}

	var privateChatsByLang map[string]int64
	var groupChatsByLang map[string]int64

	json.Unmarshal(stats.PrivateChatsByLanguage, &privateChatsByLang)
	json.Unmarshal(stats.GroupChatsByLanguage, &groupChatsByLang)

	sizeGB := float64(stats.TotalDownloadsSize) / (1024 * 1024 * 1024)

	type langStat struct {
		lang  string
		count int64
	}

	message := fmt.Sprintf("<b>stats - %s</b>\n\n", periodText)
	message += fmt.Sprintf("<b>private chats:</b> %d\n", stats.TotalPrivateChats)

	if len(privateChatsByLang) > 0 {
		message += "  languages:\n"
		langs := make([]langStat, 0, len(privateChatsByLang))
		for k, v := range privateChatsByLang {
			langs = append(langs, langStat{lang: k, count: v})
		}
		slices.SortFunc(langs, func(a, b langStat) int {
			return int(b.count - a.count)
		})
		for _, item := range langs {
			message += fmt.Sprintf("    • %s: %d\n", item.lang, item.count)
		}
	}

	message += fmt.Sprintf("\n<b>group chats:</b> %d\n", stats.TotalGroupChats)

	if len(groupChatsByLang) > 0 {
		message += "  languages:\n"
		langs := make([]langStat, 0, len(groupChatsByLang))
		for k, v := range groupChatsByLang {
			langs = append(langs, langStat{lang: k, count: v})
		}
		slices.SortFunc(langs, func(a, b langStat) int {
			return int(b.count - a.count)
		})
		for _, item := range langs {
			message += fmt.Sprintf("    • %s: %d\n", item.lang, item.count)
		}
	}

	message += fmt.Sprintf("\n<b>downloads:</b> %d\n", stats.TotalDownloads)
	message += fmt.Sprintf("<b>total size:</b> %.2f GB\n", sizeGB)

	return message, nil
}

func getStatsKeyboard() gotgbot.InlineKeyboardMarkup {
	return gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{
				{
					Text:         "1d",
					CallbackData: "stats:1d",
				},
				{
					Text:         "7d",
					CallbackData: "stats:7d",
				},
				{
					Text:         "30d",
					CallbackData: "stats:30d",
				},
				{
					Text:         "all",
					CallbackData: "stats:all",
				},
			},
		},
	}
}

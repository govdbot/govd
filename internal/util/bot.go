package util

import (
	"strconv"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/govdbot/govd/internal/localization"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func Unquote(text string) string {
	// we wont use html.EscapeString
	// because it will escape all the characters
	// and we only need to escape < and >
	// (to avoid telegram formatting issues)
	return strings.NewReplacer(
		"<", "&lt;",
		">", "&gt;",
	).Replace(text)
}

func IsUserAdmin(bot *gotgbot.Bot, user *gotgbot.User, chatID int64) bool {
	if user == nil {
		return false
	}
	if IsAnonymousAdmin(user) {
		return true
	}
	chatMember, err := bot.GetChatMember(chatID, user.Id, nil)
	if err != nil {
		return false
	}
	if chatMember == nil {
		return false
	}
	status := chatMember.GetStatus()
	switch status {
	case "creator", "administrator":
		return true
	}
	return false
}

func IsAnonymousAdmin(user *gotgbot.User) bool {
	if user == nil {
		return false
	}
	// @GroupAnonymousBot
	return user.Id == 1087968824
}

func CheckAdminPermission(b *gotgbot.Bot, ctx *ext.Context, localizer *localization.Localizer) bool {
	if !IsUserAdmin(b, ctx.EffectiveUser, ctx.EffectiveChat.Id) {
		noPermissionMessage := localizer.T(&i18n.LocalizeConfig{
			MessageID: localization.NoPermission.ID,
		})

		if ctx.CallbackQuery != nil {
			ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
				Text:      noPermissionMessage,
				ShowAlert: true,
			})
		} else if ctx.Message != nil {
			ctx.EffectiveMessage.Reply(b, noPermissionMessage, nil)
		}
		return false
	}
	return true
}

func HasHashtagEntity(msg *gotgbot.Message, entity string) bool {
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

func URLFromMessage(msg *gotgbot.Message) string {
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

func MentionUser(user *gotgbot.User) string {
	deepLink := "tg://user?id=" + strconv.FormatInt(user.Id, 10)
	return "<a href='" + deepLink + "'>" + Unquote(user.FirstName) + "</a>"
}

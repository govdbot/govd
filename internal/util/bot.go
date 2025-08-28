package util

import (
	"strconv"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
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

func MentionUser(user *gotgbot.User) string {
	deepLink := "tg://user?id=" + strconv.FormatInt(user.Id, 10)
	return "<a href='" + deepLink + "'>" + Unquote(user.FirstName) + "</a>"
}

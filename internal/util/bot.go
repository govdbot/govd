package util

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/localization"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func ChatFromContext(ctx *ext.Context) (*database.GetOrCreateChatRow, error) {
	var id int64
	var chatType database.ChatType
	var languageCode string

	switch {
	case ctx.Message != nil:
		chat := ctx.EffectiveMessage.Chat
		id = chat.Id
		if ctx.EffectiveUser != nil {
			languageCode = ctx.EffectiveUser.LanguageCode
		}
		if chat.Type == gotgbot.ChatTypePrivate {
			chatType = database.ChatTypePrivate
		} else {
			chatType = database.ChatTypeGroup
		}
	case ctx.InlineQuery != nil:
		id = ctx.InlineQuery.From.Id
		languageCode = ctx.InlineQuery.From.LanguageCode
		chatType = database.ChatTypePrivate
	case ctx.CallbackQuery != nil:
		if ctx.CallbackQuery.Message == nil {
			chatType = database.ChatTypePrivate
			id = ctx.CallbackQuery.From.Id
			languageCode = ctx.CallbackQuery.From.LanguageCode
		} else {
			chat := ctx.CallbackQuery.Message.GetChat()
			if chat.Type == gotgbot.ChatTypePrivate {
				chatType = database.ChatTypePrivate
			} else {
				chatType = database.ChatTypeGroup
			}
			languageCode = ctx.CallbackQuery.From.LanguageCode
			id = chat.Id
		}
	case ctx.MyChatMember != nil:
		chat := ctx.MyChatMember.Chat
		if chat.Type == gotgbot.ChatTypePrivate {
			chatType = database.ChatTypePrivate
		} else {
			chatType = database.ChatTypeGroup
		}
		if ctx.EffectiveUser != nil {
			languageCode = ctx.EffectiveUser.LanguageCode
		}
		id = chat.Id
	default:
		return nil, fmt.Errorf("unable to determine chat from context")
	}

	var language string
	if config.Env.AutomaticLanguageDetection {
		language = localization.GetLocaleFromCode(
			languageCode,
			config.Env.DefaultLanguage,
		)
	} else {
		language = config.Env.DefaultLanguage
	}
	res, err := database.Q().GetOrCreateChat(
		context.Background(),
		database.GetOrCreateChatParams{
			ChatID:          id,
			Type:            chatType,
			Language:        language,
			Captions:        config.Env.DefaultCaptions,
			Silent:          config.Env.DefaultSilent,
			Nsfw:            config.Env.DefaultNSFW,
			MediaAlbumLimit: config.Env.DefaultMediaAlbumLimit,
		},
	)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func HashHashtagEntity(msg *gotgbot.Message, entity string) bool {
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

func SendTypingAction(b *gotgbot.Bot, chatID int64) error {
	_, err := b.SendChatAction(chatID, "typing", nil)
	return err
}

func SendMediaAction(b *gotgbot.Bot, chatID int64, mediaType database.MediaType) {
	var action string
	switch mediaType {
	case database.MediaTypeVideo:
		action = "upload_video"
	case database.MediaTypeAudio:
		action = "upload_audio"
	case database.MediaTypePhoto:
		action = "upload_photo"
	default:
		action = "upload_document"
	}
	b.SendChatAction(chatID, action, nil)
}

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

func GetMessageFileID(msg *gotgbot.Message) string {
	switch {
	case msg.Video != nil:
		return msg.Video.FileId
	case msg.Animation != nil:
		return msg.Animation.FileId
	case msg.Photo != nil:
		return msg.Photo[len(msg.Photo)-1].FileId
	case msg.Document != nil:
		return msg.Document.FileId
	case msg.Audio != nil:
		return msg.Audio.FileId
	case msg.Voice != nil:
		return msg.Voice.FileId
	default:
		return ""
	}
}

func GetMessageFileSize(msg *gotgbot.Message) int64 {
	switch {
	case msg.Video != nil:
		return msg.Video.FileSize
	case msg.Animation != nil:
		return msg.Animation.FileSize
	case msg.Photo != nil:
		return msg.Photo[len(msg.Photo)-1].FileSize
	case msg.Document != nil:
		return msg.Document.FileSize
	case msg.Audio != nil:
		return msg.Audio.FileSize
	case msg.Voice != nil:
		return msg.Voice.FileSize
	default:
		return 0
	}
}

func IsBotAdmin(ctx *ext.Context) bool {
	chatType := ctx.EffectiveChat.Type
	if chatType != gotgbot.ChatTypePrivate {
		return false
	}
	userID := ctx.EffectiveMessage.From.Id
	return slices.Contains(config.Env.Admins, userID)
}

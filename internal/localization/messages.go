package localization

import "github.com/nicksnyder/go-i18n/v2/i18n"

var (
	Language = &i18n.Message{
		ID:    "Language",
		Other: "english",
	}
	StartMessage = &i18n.Message{
		ID:    "StartMessage",
		Other: "welcome {{.Name}} to govd, an open-source telegram bot for downloading content from various social platforms",
	}
	AddButton = &i18n.Message{
		ID:    "AddButton",
		Other: "add to a group",
	}
	ErrorMessage = &i18n.Message{
		ID:    "ErrorMessage",
		Other: "an error occurred, please try again later",
	}
	AddedToGroupMessage = &i18n.Message{
		ID:    "AddedToGroupMessage",
		Other: "thank you for adding me! use /settings command to configure the bot for this group",
	}
	SettingsButton = &i18n.Message{
		ID:    "SettingsButton",
		Other: "settings",
	}
	LanguageButton = &i18n.Message{
		ID:    "LanguageButton",
		Other: "language",
	}
	PrivateSettingsMessage = &i18n.Message{
		ID:    "PrivateSettingsMessage",
		Other: "use the buttons below to change your personal bot settings",
	}
	GroupSettingsMessage = &i18n.Message{
		ID:    "GroupSettingsMessage",
		Other: "use the buttons below to change this group's bot settings",
	}
	BackButton = &i18n.Message{
		ID:    "BackButton",
		Other: "back",
	}
	SelectLanguageMessage = &i18n.Message{
		ID:    "SelectLanguageMessage",
		Other: "select your preferred language",
	}
	CaptionsSettingsMessage = &i18n.Message{
		ID:    "CaptionsSettingsMessage",
		Other: "when enabled, adds original description to downloaded content, if available",
	}
	NsfwSettingsMessage = &i18n.Message{
		ID:    "NsfwSettingsMessage",
		Other: "when enabled, allows downloading nsfw content in this chat\n\nwarning: such content may violate telegram's terms of service and result in group restrictions",
	}
	SilentModeSettingsMessage = &i18n.Message{
		ID:    "SilentModeSettingsMessage",
		Other: "when enabled, the bot will not send error messages",
	}
	MediaAlbumSettingsMessage = &i18n.Message{
		ID:    "MediaAlbumSettingsMessage",
		Other: "select maximum number of files allowed in a single media album",
	}
	NoPermission = &i18n.Message{
		ID:    "NoPermission",
		Other: "you don't have permissions to perform this action",
	}
	CloseButton = &i18n.Message{
		ID:    "CloseButton",
		Other: "close",
	}
	MediaAlbumButton = &i18n.Message{
		ID:    "MediaAlbumButton",
		Other: "media album",
	}
	SilentModeButton = &i18n.Message{
		ID:    "SilentModeButton",
		Other: "silent mode",
	}
	CaptionsButton = &i18n.Message{
		ID:    "CaptionsButton",
		Other: "captions",
	}
	NsfwButton = &i18n.Message{
		ID:    "NsfwButton",
		Other: "nsfw",
	}
	EnabledButton = &i18n.Message{
		ID:    "EnabledButton",
		Other: "enabled",
	}
	DisabledButton = &i18n.Message{
		ID:    "DisabledButton",
		Other: "disabled",
	}
	ErrorUnavailable = &i18n.Message{
		ID:    "ErrorUnavailable",
		Other: "this content is unavailable",
	}
	ErrorTimeout = &i18n.Message{
		ID:    "ErrorTimeout",
		Other: "timeout error when downloading. try again later.",
	}
	ErrorUnsupportedImageFormat = &i18n.Message{
		ID:    "ErrorUnsupportedImageFormat",
		Other: "unsupported image format",
	}
	ErrorUnsupportedExtractorType = &i18n.Message{
		ID:    "ErrorUnsupportedExtractorType",
		Other: "unsupported extractor type",
	}
	ErrorMediaAlbumLimitExceeded = &i18n.Message{
		ID:    "ErrorMediaAlbumLimitExceeded",
		Other: "media album limit exceeds the maximum allowed for this group. change /settings to increase the limit.",
	}
	ErrorNSFWNotAllowed = &i18n.Message{
		ID:    "ErrorNSFWNotAllowed",
		Other: "this content is marked as nsfw and can't be downloaded in this group. change /settings to allow nsfw content or use the bot privately.",
	}
	ErrorInlineMediaAlbum = &i18n.Message{
		ID:    "ErrorInlineMediaAlbum",
		Other: "you can't download media albums in inline mode. use the bot in a group or private chat.",
	}
	ErrorAuthenticationNeeded = &i18n.Message{
		ID:    "ErrorAuthenticationNeeded",
		Other: "this instance is not authenticated with this service.",
	}
	ErrorFileTooLarge = &i18n.Message{
		ID:    "ErrorFileTooLarge",
		Other: "this file is too large and exceeds the maximum allowed size for this instance.",
	}
	ErrorTelegramFileTooLarge = &i18n.Message{
		ID:    "ErrorTelegramFileTooLarge",
		Other: "this file is too large for telegram and exceeds the maximum allowed size.",
	}
	ErrorDurationTooLong = &i18n.Message{
		ID:    "ErrorDurationTooLong",
		Other: "this video is too long and exceeds the maximum allowed duration for this instance.",
	}
	ErrorPaidContent = &i18n.Message{
		ID:    "ErrorPaidContent",
		Other: "this content is paid and requires a subscription to access.",
	}
)

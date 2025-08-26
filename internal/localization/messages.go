package localization

import "github.com/nicksnyder/go-i18n/v2/i18n"

var (
	StartMessage = &i18n.Message{
		ID:    "Start",
		Other: "welcome {{.Name}} to govd, an open-source telegram bot for downloading content from various social platforms",
	}
	AddButtonMessage = &i18n.Message{
		ID:    "AddButton",
		Other: "add to a group",
	}
	ErrorMessage = &i18n.Message{
		ID:    "Error",
		Other: "an error occurred, please try again later",
	}
	AddedToGroupMessage = &i18n.Message{
		ID:    "AddedToGroup",
		Other: "thank you for adding me! use the buttons below to change settings",
	}
)

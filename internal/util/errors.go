package util

type Error struct {
	Message string
}

func (err *Error) Error() string {
	return err.Message
}

var (
	ErrUnavailable              = &Error{Message: "This content is unavailable."}
	ErrNotImplemented           = &Error{Message: "This feature is not implemented."}
	ErrTimeout                  = &Error{Message: "Timeout error when downloading. Try again later."}
	ErrUnsupportedImageFormat   = &Error{Message: "Unsupported image format"}
	ErrUnsupportedExtractorType = &Error{Message: "Unsupported extractor type"}
	ErrMediaAlbumLimitExceeded  = &Error{Message: "Media album limit exceeds the maximum allowed for this group. Change /settings to increase the limit."}
	ErrNSFWNotAllowed           = &Error{Message: "This content is marked as NSFW and can't be downloaded in this group. Change /settings to allow NSFW content or use the bot privately."}
	ErrInlineMediaAlbum         = &Error{Message: "You can't download media albums in inline mode. Use the bot in a group or private chat."}
	ErrAuthenticationNeeded     = &Error{Message: "This instance is not authenticated with this service."}
	ErrFileTooLarge             = &Error{Message: "This file is too large and exceeds the maximum allowed size for this instance."}
	ErrTelegramFileTooLarge     = &Error{Message: "This file is too large for Telegram and exceeds the maximum allowed size."}
	ErrDurationTooLong          = &Error{Message: "This video is too long and exceeds the maximum allowed duration for this instance."}
	ErrPaidContent              = &Error{Message: "This content is paid and requires a subscription to access."}
)

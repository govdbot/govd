package util

import (
	"encoding/hex"
	"hash/fnv"
	"strings"

	"github.com/govdbot/govd/internal/localization"
)

type Error struct {
	ID string
}

func (err *Error) Error() string {
	return err.ID
}

var (
	ErrUnavailable                   = &Error{ID: localization.ErrorUnavailable.ID}
	ErrTimeout                       = &Error{ID: localization.ErrorTimeout.ID}
	ErrUnsupportedImageFormat        = &Error{ID: localization.ErrorUnsupportedImageFormat.ID}
	ErrUnsupportedExtractorType      = &Error{ID: localization.ErrorUnsupportedExtractorType.ID}
	ErrMediaAlbumLimitExceeded       = &Error{ID: localization.ErrorMediaAlbumLimitExceeded.ID}
	ErrMediaAlbumGlobalLimitExceeded = &Error{ID: localization.ErrorMediaAlbumGlobalLimitExceeded.ID}
	ErrGeoRestrictedContent          = &Error{ID: localization.ErrorGeoRestrictedContent.ID}
	ErrNSFWNotAllowed                = &Error{ID: localization.ErrorNSFWNotAllowed.ID}
	ErrInlineMediaAlbum              = &Error{ID: localization.ErrorInlineMediaAlbum.ID}
	ErrAuthenticationNeeded          = &Error{ID: localization.ErrorAuthenticationNeeded.ID}
	ErrFileTooLarge                  = &Error{ID: localization.ErrorFileTooLarge.ID}
	ErrTelegramFileTooLarge          = &Error{ID: localization.ErrorTelegramFileTooLarge.ID}
	ErrDurationTooLong               = &Error{ID: localization.ErrorDurationTooLong.ID}
	ErrPaidContent                   = &Error{ID: localization.ErrorPaidContent.ID}
)

func HashedError(message string) string {
	const length = 8

	h := fnv.New64a()
	h.Write([]byte(message))

	sum := h.Sum(nil)
	hexStr := hex.EncodeToString(sum)

	if length > len(hexStr) {
		return strings.ToUpper(hexStr)
	}
	return strings.ToUpper(hexStr[:length])
}

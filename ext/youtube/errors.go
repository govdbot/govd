package youtube

import "github.com/govdbot/govd/util"

var (
	ErrNotConfigured   = &util.Error{Message: "youtube extractor is not configured"}
	ErrAgeRestricted   = &util.Error{Message: "video is age-restricted and cannot be downloaded"}
	ErrNoValidFormats  = &util.Error{Message: "no valid formats found for the video"}
	ErrInvidiousNotSet = &util.Error{Message: "invidious instance url is not set"}
)

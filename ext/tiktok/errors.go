package tiktok

import "github.com/govdbot/govd/util"

var (
	ErrRegionNotSupported    = &util.Error{Message: "tiktok is geo restricted in your region, use cookies to bypass or use a VPN/proxy"}
	ErrAllMethodsFailed      = &util.Error{Message: "all methods failed"}
	ErrAwemeDetailNil        = &util.Error{Message: "aweme detail is nil, this usually means the video is private or deleted"}
	ErrURLKeyNotFound        = &util.Error{Message: "url key not found"}
	ErrUniversalDataNotFound = &util.Error{Message: "universal data not found in response"}
	ErrDefaultScopeNotFound  = &util.Error{Message: "default scope not found in response"}
	ErrItemStructNotFound    = &util.Error{Message: "item struct not found in response"}
)

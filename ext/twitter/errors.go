package twitter

import "github.com/govdbot/govd/util"

var (
	ErrURLNotFound    = &util.Error{Message: "URL not found in response"}
	ErrInvalidCookies = &util.Error{Message: "invalid cookies provided"}
	ErrTweetNotFound  = &util.Error{Message: "tweet not found or deleted"}
)

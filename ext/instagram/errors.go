package instagram

import "github.com/govdbot/govd/util"

var (
	ErrAllMethodsFailed   = &util.Error{Message: "all methods failed"}
	ErrGQLJSONNotFound    = &util.Error{Message: "failed to find JSON in response"}
	ErrGQLContextNotFound = &util.Error{Message: "failed to find context in response"}
	ErrGQLContextMismatch = &util.Error{Message: "context is not a string"}
	ErrGQLNilResponse     = &util.Error{Message: "GQL data is nil"}
	ErrGQLNilMedia        = &util.Error{Message: "GQL media is nil"}
)

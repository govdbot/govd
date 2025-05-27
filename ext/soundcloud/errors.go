package soundcloud

import "github.com/govdbot/govd/util"

var (
	ErrNoSuitableFormat = &util.Error{Message: "no suitable format found for the track"}
	ErrClientIDNotFound = &util.Error{Message: "client ID not found"}
)

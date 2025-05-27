package ninegag

import "github.com/govdbot/govd/util"

var (
	ErrNoMediaFound = &util.Error{Message: "no media found"}
	ErrNoPhotoFound = &util.Error{Message: "no photo found in post"}
	ErrNoVideoFound = &util.Error{Message: "no video found in post"}
)

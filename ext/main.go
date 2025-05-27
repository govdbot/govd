package ext

import (
	"github.com/govdbot/govd/ext/instagram"
	"github.com/govdbot/govd/ext/ninegag"
	"github.com/govdbot/govd/ext/pinterest"
	"github.com/govdbot/govd/ext/reddit"
	"github.com/govdbot/govd/ext/redgifs"
	"github.com/govdbot/govd/ext/soundcloud"
	"github.com/govdbot/govd/ext/threads"
	"github.com/govdbot/govd/ext/tiktok"
	"github.com/govdbot/govd/ext/twitter"
	"github.com/govdbot/govd/ext/youtube"
	"github.com/govdbot/govd/models"
)

var List = []*models.Extractor{
	youtube.Extractor,
	tiktok.Extractor,
	tiktok.VMExtractor,
	instagram.Extractor,
	instagram.StoriesExtractor,
	instagram.ShareURLExtractor,
	threads.Extractor,
	twitter.Extractor,
	twitter.ShortExtractor,
	pinterest.Extractor,
	pinterest.ShortExtractor,
	reddit.Extractor,
	reddit.ShortExtractor,
	ninegag.Extractor,
	redgifs.Extractor,
	soundcloud.Extractor,
	soundcloud.ShortExtractor,
}

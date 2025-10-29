package extractors

import (
	"github.com/govdbot/govd/internal/extractors/instagram"
	"github.com/govdbot/govd/internal/extractors/ninegag"
	"github.com/govdbot/govd/internal/extractors/pinterest"
	"github.com/govdbot/govd/internal/extractors/reddit"
	"github.com/govdbot/govd/internal/extractors/soundcloud"
	"github.com/govdbot/govd/internal/extractors/tiktok"
	"github.com/govdbot/govd/internal/extractors/twitter"
	"github.com/govdbot/govd/internal/extractors/youtube"
	"github.com/govdbot/govd/internal/models"
)

var Extractors = []*models.Extractor{
	tiktok.Extractor,
	tiktok.VMExtractor,
	soundcloud.Extractor,
	soundcloud.ShortExtractor,
	twitter.Extractor,
	twitter.ShortExtractor,
	instagram.Extractor,
	instagram.StoriesExtractor,
	instagram.ShareURLExtractor,
	ninegag.Extractor,
	youtube.Extractor,
	pinterest.ShortExtractor,
	pinterest.Extractor,
	reddit.Extractor,
	reddit.ShortExtractor,
}

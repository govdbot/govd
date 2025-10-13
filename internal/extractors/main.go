package extractors

import (
	"github.com/govdbot/govd/internal/extractors/soundcloud"
	"github.com/govdbot/govd/internal/extractors/tiktok"
	"github.com/govdbot/govd/internal/extractors/twitter"
	"github.com/govdbot/govd/internal/models"
)

var Extractors = []*models.Extractor{
	tiktok.Extractor,
	tiktok.VMExtractor,
	soundcloud.Extractor,
	soundcloud.ShortExtractor,
	twitter.Extractor,
	twitter.ShortExtractor,
}

package extractors

import (
	"github.com/govdbot/govd/internal/extractors/tiktok"
	"github.com/govdbot/govd/internal/models"
)

var Extractors = []*models.Extractor{
	tiktok.Extractor,
}

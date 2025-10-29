package reddit

type Child struct {
	Data *PostData `json:"data"`
}

type Data struct {
	Children []*Child `json:"children"`
}

type ResponseItem struct {
	Data *Data `json:"data"`
}

type Response []*ResponseItem

type PostData struct {
	ID            string                   `json:"id"`
	Title         string                   `json:"title"`
	IsVideo       bool                     `json:"is_video"`
	Thumbnail     string                   `json:"thumbnail"`
	Media         *Media                   `json:"media"`
	Preview       *Preview                 `json:"preview"`
	MediaMetadata map[string]MediaMetadata `json:"media_metadata"`
	SecureMedia   *Media                   `json:"secure_media"`
	Over18        bool                     `json:"over_18"`
}

type Media struct {
	Video *Video `json:"reddit_video"`
}

type Video struct {
	FallbackURL      string `json:"fallback_url"`
	HLSURL           string `json:"hls_url"`
	DashURL          string `json:"dash_url"`
	Duration         int32  `json:"duration"`
	Height           int32  `json:"height"`
	Width            int32  `json:"width"`
	ScrubberMediaURL string `json:"scrubber_media_url"`
}

type Preview struct {
	Images       []Image       `json:"images"`
	VideoPreview *VideoPreview `json:"reddit_video_preview"`
}

type Image struct {
	Source   ImageSource   `json:"source"`
	Variants ImageVariants `json:"variants"`
}

type ImageSource struct {
	URL    string `json:"url"`
	Width  int32  `json:"width"`
	Height int32  `json:"height"`
}

type ImageVariants struct {
	MP4 *MP4Variant `json:"mp4"`
}

type MP4Variant struct {
	Source ImageSource `json:"source"`
}

type VideoPreview struct {
	FallbackURL string `json:"fallback_url"`
	Duration    int32  `json:"duration"`
}

type MediaMetadata struct {
	Status string `json:"status"`
	Type   string `json:"e"`
	Media  struct {
		MP4    string `json:"mp4"`
		URL    string `json:"u"`
		Width  int64  `json:"x"`
		Height int64  `json:"y"`
	} `json:"s"`
}

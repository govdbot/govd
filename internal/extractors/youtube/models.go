package youtube

type InvResponse struct {
	Type             string            `json:"type"`
	Title            string            `json:"title"`
	VideoID          string            `json:"videoId"`
	VideoThumbnails  []*VideoThumbnail `json:"videoThumbnails"`
	Description      string            `json:"description"`
	DescriptionHTML  string            `json:"descriptionHtml"`
	Published        int               `json:"published"`
	ViewCount        int               `json:"viewCount"`
	LikeCount        int               `json:"likeCount"`
	DislikeCount     int               `json:"dislikeCount"`
	Paid             bool              `json:"paid"`
	Premium          bool              `json:"premium"`
	IsFamilyFriendly bool              `json:"isFamilyFriendly"`
	AllowedRegions   []string          `json:"allowedRegions"`
	Genre            string            `json:"genre"`
	Author           string            `json:"author"`
	AuthorID         string            `json:"authorId"`
	AuthorURL        string            `json:"authorUrl"`
	AuthorVerified   bool              `json:"authorVerified"`
	SubCountText     string            `json:"subCountText"`
	LengthSeconds    int32             `json:"lengthSeconds"`
	AllowRatings     bool              `json:"allowRatings"`
	Rating           int               `json:"rating"`
	IsListed         bool              `json:"isListed"`
	LiveNow          bool              `json:"liveNow"`
	IsPostLiveDvr    bool              `json:"isPostLiveDvr"`
	IsUpcoming       bool              `json:"isUpcoming"`
	DashURL          string            `json:"dashUrl"`
	AdaptiveFormats  []*AdaptiveFormat `json:"adaptiveFormats"`
	FormatStreams    []*FormatStream   `json:"formatStreams"`
	Error            string            `json:"error,omitempty"`
}

type VideoThumbnail struct {
	Quality string `json:"quality"`
	URL     string `json:"url"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
}

type AdaptiveFormat struct {
	Init            string `json:"init"`
	Index           string `json:"index"`
	Bitrate         string `json:"bitrate"`
	URL             string `json:"url"`
	Itag            string `json:"itag"`
	Type            string `json:"type"`
	Clen            string `json:"clen"`
	Lmt             string `json:"lmt"`
	ProjectionType  string `json:"projectionType"`
	Container       string `json:"container,omitempty"`
	Encoding        string `json:"encoding,omitempty"`
	AudioQuality    string `json:"audioQuality,omitempty"`
	AudioSampleRate int    `json:"audioSampleRate,omitempty"`
	AudioChannels   int    `json:"audioChannels,omitempty"`
	Fps             int    `json:"fps,omitempty"`
	Size            string `json:"size,omitempty"`
	Resolution      string `json:"resolution,omitempty"`
	QualityLabel    string `json:"qualityLabel,omitempty"`
}

type FormatStream struct {
	URL          string `json:"url"`
	Itag         string `json:"itag"`
	Type         string `json:"type"`
	Quality      string `json:"quality"`
	Bitrate      string `json:"bitrate"`
	Fps          int    `json:"fps"`
	Size         string `json:"size"`
	Resolution   string `json:"resolution"`
	QualityLabel string `json:"qualityLabel"`
	Container    string `json:"container"`
	Encoding     string `json:"encoding"`
}

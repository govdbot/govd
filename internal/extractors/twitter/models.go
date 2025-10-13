package twitter

type APIResponse struct {
	Data struct {
		TweetResult struct {
			Result *TweetResult `json:"result"`
		} `json:"tweetResult"`
	} `json:"data"`
}

type TweetResult struct {
	Tweet       *TweetResultData `json:"tweet"`
	Legacy      *Tweet           `json:"legacy"`
	RestID      string           `json:"rest_id"`
	Core        *Core            `json:"core"`
	Views       *ViewsInfo       `json:"views"`
	Source      string           `json:"source"`
	EditControl *EditInfo        `json:"edit_control"`
	TypeName    string           `json:"__typename"`
}

type TweetResultData struct {
	Legacy      *Tweet     `json:"legacy"`
	RestID      string     `json:"rest_id"`
	Core        *Core      `json:"core"`
	Views       *ViewsInfo `json:"views"`
	Source      string     `json:"source"`
	EditControl *EditInfo  `json:"edit_control"`
	TypeName    string     `json:"__typename"`
}

type EditInfo struct {
	EditTweetIDs   []string `json:"edit_tweet_ids"`
	EditableUntil  string   `json:"editable_until_msecs"`
	IsEditEligible bool     `json:"is_edit_eligible"`
	EditsRemaining string   `json:"edits_remaining"`
}

type ViewsInfo struct {
	Count string `json:"count"`
	State string `json:"state"`
}

type Core struct {
	UserResults struct {
		Result struct {
			TypeName string      `json:"__typename"`
			RestID   string      `json:"rest_id"`
			Legacy   *UserLegacy `json:"legacy"`
		} `json:"result"`
	} `json:"user_results"`
}

type UserLegacy struct {
	ScreenName           string `json:"screen_name"`
	Name                 string `json:"name"`
	ProfileImageURLHTTPS string `json:"profile_image_url_https"`
	CreatedAt            string `json:"created_at"`
}

type Tweet struct {
	FullText          string            `json:"full_text"`
	ExtendedEntities  *ExtendedEntities `json:"extended_entities"`
	Entities          *ExtendedEntities `json:"entities"`
	CreatedAt         string            `json:"created_at"`
	ID                string            `json:"id_str"`
	BookmarkCount     int               `json:"bookmark_count"`
	FavoriteCount     int               `json:"favorite_count"`
	ReplyCount        int               `json:"reply_count"`
	RetweetCount      int               `json:"retweet_count"`
	QuoteCount        int               `json:"quote_count"`
	PossiblySensitive bool              `json:"possibly_sensitive"`
	ConversationID    string            `json:"conversation_id_str"`
	Lang              string            `json:"lang"`
	UserIDStr         string            `json:"user_id_str"`
}

type ExtendedEntities struct {
	Media []*MediaEntity `json:"media"`
}

type MediaEntity struct {
	Type              string             `json:"type"`
	MediaURLHTTPS     string             `json:"media_url_https"`
	ExpandedURL       string             `json:"expanded_url"`
	URL               string             `json:"url"`
	DisplayURL        string             `json:"display_url"`
	IDStr             string             `json:"id_str"`
	MediaKey          string             `json:"media_key"`
	VideoInfo         *VideoInfo         `json:"video_info"`
	Sizes             *MediaSizes        `json:"sizes"`
	OriginalInfo      *OriginalInfo      `json:"original_info"`
	MediaAvailability *MediaAvailability `json:"ext_media_availability"`
}

type MediaAvailability struct {
	Status string `json:"status"`
}

type MediaSizes struct {
	Large  SizeInfo `json:"large"`
	Medium SizeInfo `json:"medium"`
	Small  SizeInfo `json:"small"`
	Thumb  SizeInfo `json:"thumb"`
}

type SizeInfo struct {
	H      int    `json:"h"`
	W      int    `json:"w"`
	Resize string `json:"resize"`
}

type OriginalInfo struct {
	Height int `json:"height"`
	Width  int `json:"width"`
}

type VideoInfo struct {
	DurationMillis int       `json:"duration_millis"`
	Variants       []Variant `json:"variants"`
	AspectRatio    []int     `json:"aspect_ratio"`
}

type Variant struct {
	Bitrate     int    `json:"bitrate"`
	ContentType string `json:"content_type"`
	URL         string `json:"url"`
}

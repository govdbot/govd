package instagram

type GraphQLResponse struct {
	Data   *GraphQLData `json:"data"`
	Status string       `json:"status"`
}

type GraphQLData struct {
	ShortcodeMedia *Media `json:"xdt_shortcode_media"`
}

type ContextJSON struct {
	Context *Context `json:"context"`
	GqlData *GqlData `json:"gql_data"`
}

type GqlData struct {
	ShortcodeMedia *Media `json:"shortcode_media"`
}

type EdgeMediaToCaption struct {
	Edges []*Edges `json:"edges"`
}

type EdgeNode struct {
	Node *Media `json:"node"`
}

type EdgeSidecarToChildren struct {
	Edges []*EdgeNode `json:"edges"`
}

type Dimensions struct {
	Height int32 `json:"height"`
	Width  int32 `json:"width"`
}

type DisplayResources struct {
	ConfigHeight int32  `json:"config_height"`
	ConfigWidth  int32  `json:"config_width"`
	Src          string `json:"src"`
}

type Node struct {
	Text string `json:"text"`
}

type Edges struct {
	Node *Node `json:"node"`
}
type Media struct {
	Typename              string                 `json:"__typename"`
	CommenterCount        int                    `json:"commenter_count"`
	Dimensions            *Dimensions            `json:"dimensions"`
	DisplayResources      []*DisplayResources    `json:"display_resources"`
	EdgeMediaToCaption    *EdgeMediaToCaption    `json:"edge_media_to_caption"`
	EdgeSidecarToChildren *EdgeSidecarToChildren `json:"edge_sidecar_to_children"`
	DisplayURL            string                 `json:"display_url"`
	ID                    string                 `json:"id"`
	IsVideo               bool                   `json:"is_video"`
	MediaPreview          string                 `json:"media_preview"`
	Shortcode             string                 `json:"shortcode"`
	TakenAtTimestamp      int                    `json:"taken_at_timestamp"`
	Title                 string                 `json:"title"`
	VideoURL              string                 `json:"video_url"`
	VideoViewCount        int                    `json:"video_view_count"`
}

type Posts struct {
	Src    string `json:"src"`
	Srcset string `json:"srcset"`
}

type Context struct {
	AltText               string `json:"alt_text"`
	Caption               string `json:"caption"`
	CaptionTitleLinkified string `json:"caption_title_linkified"`
	DisplaySrc            string `json:"display_src"`
	DisplaySrcset         string `json:"display_srcset"`
	IsIgtv                bool   `json:"is_igtv"`
	LikesCount            int    `json:"likes_count"`
	Media                 *Media `json:"media"`
	MediaPermalink        string `json:"media_permalink"`
	RequestID             string `json:"request_id"`
	Shortcode             string `json:"shortcode"`
	Title                 string `json:"title"`
	Type                  string `json:"type"`
	Username              string `json:"username"`
	Verified              bool   `json:"verified"`
	VideoViews            int    `json:"video_views"`
}

type IGramResponse struct {
	Items   []*IGramMedia `json:"items"`
	Success *bool         `json:"success"`
}

type IGramMedia struct {
	URL       []*IGramMediaURL `json:"url"`
	Thumb     string           `json:"thumb"`
	Hosting   string           `json:"hosting"`
	Timestamp int              `json:"timestamp"`
}

type IGramMediaURL struct {
	URL  string `json:"url"`
	Name string `json:"name"`
	Type string `json:"type"`
	Ext  string `json:"ext"`
}

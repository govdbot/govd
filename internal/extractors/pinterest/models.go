package pinterest

type PinResponse struct {
	ResourceResponse struct {
		Data PinData `json:"data"`
	} `json:"resource_response"`
}

type PinData struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Images       *Images   `json:"images,omitempty"`
	Videos       *Videos   `json:"videos,omitempty"`
	StoryPinData *StoryPin `json:"story_pin_data,omitempty"`
	Embed        *Embed    `json:"embed,omitempty"`
}

type Images struct {
	Orig *ImageObject `json:"orig"`
}

type ImageObject struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type Videos struct {
	VideoList map[string]*VideoObject `json:"video_list"`
}

type VideoObject struct {
	URL       string `json:"url"`
	Width     int32  `json:"width"`
	Height    int32  `json:"height"`
	Duration  int32  `json:"duration"`
	Thumbnail string `json:"thumbnail"`
}

type StoryPin struct {
	Pages []Page `json:"pages"`
}

type Page struct {
	Blocks []Block `json:"blocks"`
	Image  *struct {
		Images struct {
			Originals *ImageObject `json:"originals"`
		} `json:"images"`
	} `json:"image,omitempty"`
}

type Block struct {
	BlockType int     `json:"block_type"`
	Video     *Videos `json:"video,omitempty"`
}

type Embed struct {
	Type string `json:"type"`
	Src  string `json:"src"`
}

package tiktok

type Response struct {
	Items      []*Item `json:"items"`
	StatusCode int     `json:"status_code"`
	StatusMsg  string  `json:"status_msg"`
}

type Item struct {
	VideoInfo     *VideoInfo     `json:"video_info"`
	ImagePostInfo *ImagePostInfo `json:"image_post_info"`
	MusicInfo     *MusicInfo     `json:"music_info"`
	AwemeType     int            `json:"aweme_type"`
	Desc          string         `json:"desc"`
	ID            int64          `json:"id"`
	IDStr         string         `json:"id_str"`
}

type Meta struct {
	Bitrate  int `json:"bitrate"`
	Duration int `json:"duration"`
	Height   int `json:"height"`
	Ratio    int `json:"ratio"`
	Width    int `json:"width"`
}

type VideoInfo struct {
	URI     string   `json:"uri"`
	URLList []string `json:"url_list"`
	Meta    *Meta    `json:"meta"`
}

type MusicInfo struct {
	ID     int64  `json:"id"`
	IDStr  string `json:"id_str"`
	Title  string `json:"title"`
	Author string `json:"author"`
}

type ImagePostInfo struct {
	Images []*Image `json:"images"`
}

type Image struct {
	DisplayImage *DisplayImage `json:"display_image"`
}

type DisplayImage struct {
	URLList []string `json:"url_list"`
}

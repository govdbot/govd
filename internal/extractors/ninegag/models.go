package ninegag

type Response struct {
	Meta *Meta `json:"meta"`
	Data *Data `json:"data"`
}

type Meta struct {
	Timestamp    int    `json:"timestamp"`
	Status       string `json:"status"`
	Sid          string `json:"sid"`
	ErrorMessage string `json:"errorMessage"`
}

type Media struct {
	Width    int32  `json:"width"`
	Height   int32  `json:"height"`
	URL      string `json:"url"`
	HasAudio int    `json:"hasAudio"`
	Duration int32  `json:"duration"`
	Vp8URL   string `json:"vp8Url"`
	H265URL  string `json:"h265Url"`
	Vp9URL   string `json:"vp9Url"`
	Av1URL   string `json:"av1Url"`
}

type Post struct {
	ID          string            `json:"id"`
	URL         string            `json:"url"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Type        string            `json:"type"`
	Nsfw        int               `json:"nsfw"`
	Images      map[string]*Media `json:"images"`
}

type Data struct {
	Post *Post `json:"post"`
}

package soundcloud

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type Format struct {
	Protocol string `json:"protocol"`
	MimeType string `json:"mime_type"`
}

type Transcoding struct {
	URL    string `json:"url"`
	Preset string `json:"preset"`
	Format Format `json:"format"`
}

type Media struct {
	Transcodings []*Transcoding `json:"transcodings"`
}

type Track struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	ArtworkURL   string `json:"artwork_url"`
	User         *User  `json:"user"`
	Media        *Media `json:"media"`
	FullDuration int32  `json:"full_duration"`
}

type TrackManifest struct {
	URL string `json:"url"`
}

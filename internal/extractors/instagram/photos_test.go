package instagram

import (
	"github.com/govdbot/govd/internal/database"
	"testing"
)

func TestPhotoMetadataAndMixedAlbums(t *testing.T) {
	photo := &ytdlpMedia{Thumbnail: "https://scontent.cdninstagram.com/image.jpg"}
	video := &ytdlpMedia{URL: "https://scontent.cdninstagram.com/video.mp4", Duration: 12}
	ctx := testContext()
	media, err := parseYTDLPMedia(ctx, &ytdlpMedia{Entries: []*ytdlpMedia{photo, video}})
	if err != nil || len(media.Items) != 2 || media.Items[0].Formats[0].Type != database.MediaTypePhoto || media.Items[1].Formats[0].Type != database.MediaTypeVideo {
		t.Fatalf("mixed album: %v", err)
	}
	ctx.ContentURL = "https://www.instagram.com/reel/B/"
	if _, err := parseYTDLPMedia(ctx, photo); err == nil {
		t.Fatal("video poster accepted as reel")
	}
	ctx = testContext()
	if _, err := parseYTDLPMedia(ctx, &ytdlpMedia{Thumbnail: photo.Thumbnail, Duration: 12}); err == nil {
		t.Fatal("unavailable video converted to photo")
	}
}
func TestMissingSidecarNodesCannotCreateEmptyItems(t *testing.T) {
	media, err := ParseGQLMedia(testContext(), &Media{Typename: "GraphSidecar", EdgeSidecarToChildren: &EdgeSidecarToChildren{Edges: []*EdgeNode{nil, {Node: nil}, {Node: &Media{Typename: "GraphImage", DisplayURL: "https://scontent.cdninstagram.com/image.jpg"}}}}})
	if err != nil || len(media.Items) != 1 || len(media.Items[0].Formats) != 1 {
		t.Fatalf("nil sidecar: %v", err)
	}
}

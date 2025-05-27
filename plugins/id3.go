package plugins

import (
	"os"

	"github.com/govdbot/govd/models"

	"fmt"

	"github.com/bogem/id3v2/v2"
)

func SetID3(
	media *models.DownloadedMedia,
	downloadConfig *models.DownloadConfig,
) error {
	tag, err := id3v2.Open(
		media.FilePath,
		id3v2.Options{},
	)
	if err != nil {
		return fmt.Errorf("failed to open ID3 tag: %w", err)
	}
	defer tag.Close()

	tag.SetTitle(media.Media.Format.Title)
	tag.SetArtist(media.Media.Format.Artist)

	if media.ThumbnailFilePath != "" {
		imageData, err := os.ReadFile(media.ThumbnailFilePath)
		if err != nil {
			return fmt.Errorf("failed to read image file: %w", err)
		}

		pic := id3v2.PictureFrame{
			Encoding:    id3v2.EncodingUTF8,
			MimeType:    "image/jpeg",
			PictureType: id3v2.PTFrontCover,
			Description: "Front Cover",
			Picture:     imageData,
		}
		tag.AddAttachedPicture(pic)
	}

	if err := tag.Save(); err != nil {
		return fmt.Errorf("failed to save ID3 tag: %w", err)
	}

	return nil
}

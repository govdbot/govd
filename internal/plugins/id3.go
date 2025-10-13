package plugins

import (
	"os"

	"github.com/govdbot/govd/internal/models"

	"fmt"

	"github.com/bogem/id3v2/v2"
)

var ID3 = &models.Plugin{
	ID: "id3",
	RunFunc: func(ctx *models.ExtractorContext, format *models.DownloadedFormat) error {
		if format.FilePath == "" {
			return fmt.Errorf("file path is empty")
		}
		tag, err := id3v2.Open(
			format.FilePath,
			id3v2.Options{},
		)
		if err != nil {
			return fmt.Errorf("failed to open ID3 tag: %w", err)
		}
		defer tag.Close()

		tag.SetTitle(format.Format.Title)
		tag.SetArtist(format.Format.Artist)

		if format.ThumbnailFilePath != "" {
			imageData, err := os.ReadFile(format.ThumbnailFilePath)
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
	},
}

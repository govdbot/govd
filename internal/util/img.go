package util

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"os"
	"slices"

	_ "image/gif" // register GIF decoder
	_ "image/png" // register PNG decoder

	"github.com/govdbot/govd/internal/logger"
	_ "github.com/strukturag/libheif/go/heif" // register HEIF decoder
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP decoder
)

var (
	jpegHeader = []byte{0xFF, 0xD8, 0xFF}
	pngHeader  = []byte{0x89, 0x50, 0x4E, 0x47}
	gifHeader  = []byte{0x47, 0x49, 0x46}
	riffHeader = []byte{0x52, 0x49, 0x46, 0x46}
	webpHeader = []byte{0x57, 0x45, 0x42, 0x50}
)

func ImgToJPEG(
	file io.ReadSeeker,
	outputPath string,
	resize int,
) error {
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	img, err := DecodeImage(file, resize)
	if err != nil {
		os.Remove(outputPath)
		return err
	}
	if img == nil {
		// already a jpeg
		if _, err = io.Copy(outputFile, file); err != nil {
			os.Remove(outputPath)
			return fmt.Errorf("failed to copy image: %w", err)
		}
		return nil
	}
	err = jpeg.Encode(outputFile, img, nil)
	if err != nil {
		os.Remove(outputPath)
		return fmt.Errorf("failed to encode image: %w", err)
	}
	return nil
}

func DecodeImage(
	file io.ReadSeeker,
	resize int,
) (image.Image, error) {
	format, err := DetectImageFormat(file)
	if err != nil {
		return nil, err
	}
	logger.L.Debugf("detected image format: %s", format)

	if format == "jpeg" {
		if _, err = file.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("failed to reset file position: %w", err)
		}
		if resize > 0 {
			img, err := jpeg.Decode(file)
			if err != nil {
				return nil, fmt.Errorf("failed to decode image: %w", err)
			}
			dst := image.NewRGBA(image.Rect(0, 0, resize, resize))
			draw.NearestNeighbor.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
			return dst, nil
		}
		// already a jpeg, no need to decode
		return nil, nil
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to reset file position: %w", err)
	}
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}
	return img, nil
}

func DetectImageFormat(file io.ReadSeeker) (string, error) {
	header := make([]byte, 12)

	_, err := file.Read(header)
	if err != nil {
		return "", fmt.Errorf("failed to read file header: %w", err)
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("failed to reset file position: %w", err)
	}
	if len(header) < 12 {
		return "", fmt.Errorf("file header too short: %d bytes", len(header))
	}
	if bytes.HasPrefix(header, jpegHeader) {
		return "jpeg", nil
	}
	if bytes.HasPrefix(header, pngHeader) {
		return "png", nil
	}
	if bytes.HasPrefix(header, gifHeader) {
		return "gif", nil
	}
	if isHEIF(header) {
		return "heif", nil
	}
	if bytes.HasPrefix(header, riffHeader) {
		if bytes.Equal(header[8:12], webpHeader) {
			return "webp", nil
		}
		return "", ErrUnsupportedImageFormat
	}

	return "", ErrUnsupportedImageFormat
}

func isHEIF(header []byte) bool {
	isHeifHeader := header[0] == 0x00 && header[1] == 0x00 &&
		header[2] == 0x00 && (header[3] == 0x18 || header[3] == 0x1C) &&
		bytes.Equal(header[4:8], []byte("ftyp"))
	if !isHeifHeader {
		return false
	}
	heifBrands := []string{"heic", "heix", "mif1", "msf1"}
	brand := string(header[8:12])

	return slices.Contains(heifBrands, brand)
}

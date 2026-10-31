package util

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"os"
	"slices"

	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/models"
	"golang.org/x/image/draw"

	_ "image/gif" // register GIF decoder
	_ "image/png" // register PNG decoder

	_ "golang.org/x/image/webp" // register WebP decoder
)

var (
	jpegHeader = []byte{0xFF, 0xD8, 0xFF}
	pngHeader  = []byte{0x89, 0x50, 0x4E, 0x47}
	gifHeader  = []byte{0x47, 0x49, 0x46}
	riffHeader = []byte{0x52, 0x49, 0x46, 0x46}
	webpHeader = []byte{0x57, 0x45, 0x42, 0x50}
)

type imageBounds struct {
	W int32
	H int32
}

func ImgToJPEG(
	file io.ReadSeeker,
	outputPath string,
	resize int,
) (*imageBounds, error) {
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	img, isJPEG, err := DecodeImage(file, resize)
	if err != nil {
		os.Remove(outputPath)
		return nil, err
	}
	bounds := &imageBounds{
		W: int32(img.Bounds().Dx()),
		H: int32(img.Bounds().Dy()),
	}
	logger.L.Debugf("image dimensions: %dx%d", bounds.W, bounds.H)
	if isJPEG {
		if _, err = io.Copy(outputFile, file); err != nil {
			os.Remove(outputPath)
			return nil, fmt.Errorf("failed to copy image: %w", err)
		}
		return bounds, nil
	}
	err = jpeg.Encode(outputFile, img, nil)
	if err != nil {
		os.Remove(outputPath)
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	return bounds, nil
}

func DecodeImage(
	file io.ReadSeeker,
	resize int,
) (image.Image, bool, error) {
	defer file.Seek(0, io.SeekStart)

	format, err := DetectImageFormat(file)
	if err != nil {
		return nil, false, err
	}
	logger.L.Debugf("detected image format: %s", format)
	info, _, err := image.DecodeConfig(file)
	if err != nil {
		return nil, false, fmt.Errorf("invalid image header: %w", err)
	}
	if info.Width <= 0 || info.Height <= 0 || info.Width > 12000 || info.Height > 12000 || int64(info.Width)*int64(info.Height) > 25000000 {
		return nil, false, fmt.Errorf("image dimensions exceed safety limit")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, false, err
	}

	if format == "jpeg" {
		img, err := jpeg.Decode(file)
		if err != nil {
			return nil, false, fmt.Errorf("failed to decode image: %w", err)
		}
		if resize > 0 {
			dst := image.NewRGBA(image.Rect(0, 0, resize, resize))
			draw.NearestNeighbor.Scale(
				dst, dst.Bounds(),
				img, img.Bounds(),
				draw.Over, nil,
			)
			return dst, false, nil
		}
		return img, true, nil
	}
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, false, fmt.Errorf("failed to decode image: %w", err)
	}
	if resize > 0 {
		bounds := img.Bounds()
		w, h := bounds.Dx(), bounds.Dy()
		if w > h {
			h = max(1, h*resize/w)
			w = resize
		} else {
			w = max(1, w*resize/h)
			h = resize
		}
		dst := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.NearestNeighbor.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
		img = dst
	}
	return img, false, nil
}

func DetectImageFormat(file io.ReadSeeker) (models.ImageFormat, error) {
	// reset file position after detection
	defer file.Seek(0, io.SeekStart)

	header := make([]byte, 128)

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
		return models.ImageFormatJPEG, nil
	}
	if bytes.HasPrefix(header, pngHeader) {
		return models.ImageFormatPNG, nil
	}
	if bytes.HasPrefix(header, gifHeader) {
		return models.ImageFormatGIF, nil
	}
	if isHEIF(header) {
		return models.ImageFormatHEIF, nil
	}
	if bytes.HasPrefix(header, riffHeader) {
		if bytes.Equal(header[8:12], webpHeader) {
			return models.ImageFormatWEBP, nil
		}
		logger.L.Debugf("unknown image header: %x", header)
		return "", ErrUnsupportedImageFormat
	}

	logger.L.Debugf("unknown image header: %x", header)
	return "", ErrUnsupportedImageFormat
}

func isHEIF(header []byte) bool {
	isHeif := header[0] == 0x00 && header[1] == 0x00 &&
		header[2] == 0x00 && (header[3] == 0x18 || header[3] == 0x1C) &&
		bytes.Equal(header[4:8], []byte("ftyp"))
	if !isHeif {
		return false
	}
	heifBrands := []string{"heic", "heix", "mif1", "msf1"}
	brand := string(header[8:12])

	return slices.Contains(heifBrands, brand)
}

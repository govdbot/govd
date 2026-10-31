package libav

import ffmpeg "github.com/u2takey/ffmpeg-go"

// Local media only: do not interpret downloaded bytes as HLS/DASH/concat
// instructions that can open arbitrary URLs or local paths.
func inputOptions() ffmpeg.KwArgs {
	return ffmpeg.KwArgs{
		"protocol_whitelist": "file,pipe",
		"format_whitelist":   "mov,matroska,webm,mp3,aac,ogg,flac,wav,mpegts,mpeg,mpegvideo,h264,hevc,avi,flv,image2,jpeg_pipe,png_pipe,webp_pipe,gif,heif,avif",
		"threads":            2,
	}
}

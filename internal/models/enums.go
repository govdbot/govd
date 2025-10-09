package models

type FileType string

const (
	FileTypeDocument FileType = "document"
	FileTypePhoto    FileType = "photo"
	FileTypeVideo    FileType = "video"
	FileTypeAudio    FileType = "audio"
)

type FileExtension string

const (
	FileExtensionMP4  FileExtension = "mp4"
	FileExtensionWEBM FileExtension = "webm"
	FileExtensionMP3  FileExtension = "mp3"
	FileExtensionM4A  FileExtension = "m4a"
	FileExtensionFLAC FileExtension = "flac"
	FileExtensionOGG  FileExtension = "oga"
	FileExtensionJPEG FileExtension = "jpeg"
	FileExtensionWEBP FileExtension = "webp"
	FileExtensionJPG  FileExtension = "jpg"
	FileExtensionGIF  FileExtension = "gif"
	FileExtensionOGV  FileExtension = "ogv"
	FileExtensionAVI  FileExtension = "avi"
	FileExtensionMKV  FileExtension = "mkv"
	FileExtensionMOV  FileExtension = "mov"
)

type ImageFormat string

const (
	ImageFormatJPEG ImageFormat = "jpeg"
	ImageFormatPNG  ImageFormat = "png"
	ImageFormatGIF  ImageFormat = "gif"
	ImageFormatHEIF ImageFormat = "heif"
)

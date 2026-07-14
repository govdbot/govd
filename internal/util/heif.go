//go:build !lint

package util

// HEIF/HEIC image decoding is provided by the host's image libraries.
// To keep the build free of CGO/libheif dependencies (pure Go), we no
// longer register the external HEIF decoder here. HEIC thumbnails are
// skipped; all other formats still decode via the standard library.

//go:build !lint

package util

import (
	_ "github.com/strukturag/libheif/go/heif" // register HEIF decoder
)

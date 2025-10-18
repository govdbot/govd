package decrypt

import (
	"crypto/aes"
	"fmt"
	"path/filepath"
	"strings"
)

func calculateSegmentIV(baseIV []byte, mediaSequence int) []byte {
	iv := make([]byte, len(baseIV))
	copy(iv, baseIV)
	seqNum := uint32(mediaSequence)

	carry := uint32(0)

	for i := 15; i >= 12; i-- {
		sum := uint32(iv[i]) + ((seqNum >> (8 * (15 - i))) & 0xFF) + carry
		iv[i] = byte(sum & 0xFF)
		carry = sum >> 8
	}
	for i := 11; i >= 0 && carry > 0; i-- {
		sum := uint32(iv[i]) + carry
		iv[i] = byte(sum & 0xFF)
		carry = sum >> 8
	}

	return iv
}

func removePKCS7Padding(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data is empty")
	}
	paddingLength := int(data[len(data)-1])
	if paddingLength == 0 || paddingLength > aes.BlockSize {
		return nil, fmt.Errorf("invalid padding length: %d", paddingLength)
	}
	if paddingLength > len(data) {
		return nil, fmt.Errorf("padding length (%d) exceeds data length (%d)", paddingLength, len(data))
	}
	for i := len(data) - paddingLength; i < len(data); i++ {
		if data[i] != byte(paddingLength) {
			return nil, fmt.Errorf("invalid padding at position %d", i)
		}
	}
	return data[:len(data)-paddingLength], nil
}

func generateDecryptedFilename(originalPath string) string {
	dir := filepath.Dir(originalPath)
	filename := filepath.Base(originalPath)
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)
	return filepath.Join(dir, name+"_decrypted"+ext)
}

func isValidAESKey(key []byte) bool {
	return len(key) == 16
}

func isValidIV(iv []byte) bool {
	return len(iv) == 16
}

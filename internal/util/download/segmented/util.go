package segmented

import (
	"crypto/aes"
	"errors"
	"fmt"
)

// removes PKCS#7 padding from decrypted data
func removePKCS7Padding(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("data is empty")
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

// calculates the IV for a specific segment using media sequence number
// HLS specification: each segment uses base IV + media sequence number
func calculateSegmentIV(baseIV []byte, mediaSequence int) []byte {
	iv := make([]byte, len(baseIV))
	copy(iv, baseIV)

	// convert media sequence to 32-bit unsigned integer
	seqNum := uint32(mediaSequence)

	// add media sequence to the last 4 bytes of IV (big-endian)
	// this ensures proper overflow handling according to HLS spec
	carry := uint32(0)

	// start from the least significant byte and work backwards
	for i := 15; i >= 12; i-- {
		sum := uint32(iv[i]) + ((seqNum >> (8 * (15 - i))) & 0xFF) + carry
		iv[i] = byte(sum & 0xFF)
		carry = sum >> 8
	}
	// handle any remaining carry into the upper bytes
	for i := 11; i >= 0 && carry > 0; i-- {
		sum := uint32(iv[i]) + carry
		iv[i] = byte(sum & 0xFF)
		carry = sum >> 8
	}

	return iv
}

func isValidAESKey(key []byte) bool {
	return len(key) == 16
}

func isValidIV(iv []byte) bool {
	return len(iv) == 16
}

func generateZeroIV() []byte {
	return make([]byte, 16)
}

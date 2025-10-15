package segmented

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"os"
)

func (sd *SegmentedDownloader) decryptSegments(segments []string) error {
	key := sd.decryptionKey.Key
	iv := sd.decryptionKey.IV
	mediaSequence := sd.decryptionKey.MediaSequence

	if !isValidAESKey(key) {
		return fmt.Errorf("invalid key: expected 16 bytes, got %d", len(key))
	}
	if !isValidAESKey(iv) {
		return fmt.Errorf("invalid IV: expected 16 bytes, got %d", len(iv))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("failed to create AES cipher: %w", err)
	}
	for i, segment := range segments {
		err := decryptSegmentFile(segment, block, iv, mediaSequence+i)
		if err != nil {
			return fmt.Errorf("failed to decrypt segment %s: %w", segment, err)
		}
	}

	return nil
}

func decryptSegmentFile(
	segmentPath string,
	block cipher.Block,
	baseIV []byte,
	segmentSequence int,
) error {
	encryptedData, err := os.ReadFile(segmentPath)
	if err != nil {
		return fmt.Errorf("failed to read segment file: %w", err)
	}
	if len(encryptedData) == 0 {
		return errors.New("segment file is empty")
	}
	if len(encryptedData)%aes.BlockSize != 0 {
		return errors.New("encrypted data length is not a multiple of block size")
	}
	iv := calculateSegmentIV(baseIV, segmentSequence)
	mode := cipher.NewCBCDecrypter(block, iv)
	decryptedData := make([]byte, len(encryptedData))
	mode.CryptBlocks(decryptedData, encryptedData)
	unpaddedData, err := removePKCS7Padding(decryptedData)
	if err != nil {
		return fmt.Errorf("failed to remove padding: %w", err)
	}

	// overwrite original file
	if err := os.WriteFile(segmentPath, unpaddedData, 0644); err != nil {
		return fmt.Errorf("failed to write decrypted file: %w", err)
	}

	return nil
}

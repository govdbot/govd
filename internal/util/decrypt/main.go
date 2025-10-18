package decrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"io"
	"os"
)

func DecryptSegments(
	segments []string,
	key []byte,
	iv []byte,
	mediaSequence int,
) error {
	if !isValidAESKey(key) {
		return fmt.Errorf("invalid key: expected 16 bytes, got %d", len(key))
	}
	if !isValidIV(iv) {
		return fmt.Errorf("invalid IV: expected 16 bytes, got %d", len(iv))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("failed to create AES cipher: %w", err)
	}
	for i, segmentPath := range segments {
		if err := decryptSegment(segmentPath, block, iv, mediaSequence+i); err != nil {
			return fmt.Errorf("failed to decrypt segment %s: %w", segmentPath, err)
		}
	}

	return nil
}

func decryptSegment(
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
		return fmt.Errorf("segment file is empty")
	}
	if len(encryptedData)%aes.BlockSize != 0 {
		return fmt.Errorf("encrypted data length is not a multiple of block size")
	}
	iv := calculateSegmentIV(baseIV, segmentSequence)
	mode := cipher.NewCBCDecrypter(block, iv)
	decryptedData := make([]byte, len(encryptedData))
	mode.CryptBlocks(decryptedData, encryptedData)
	unpaddedData, err := removePKCS7Padding(decryptedData)
	if err != nil {
		return fmt.Errorf("failed to remove padding: %w", err)
	}
	outputPath := generateDecryptedFilename(segmentPath)
	if err := os.WriteFile(outputPath, unpaddedData, 0644); err != nil {
		return fmt.Errorf("failed to write decrypted file: %w", err)
	}

	return nil
}

func DecryptSegmentsInPlace(
	segments []string,
	key []byte,
	iv []byte,
	mediaSequence int,
) error {
	if !isValidAESKey(key) {
		return fmt.Errorf("invalid key: expected 16 bytes, got %d", len(key))
	}
	if !isValidIV(iv) {
		return fmt.Errorf("invalid IV: expected 16 bytes, got %d", len(iv))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("failed to create AES cipher: %w", err)
	}
	for i, segmentPath := range segments {
		if err := decryptSegmentInPlace(segmentPath, block, iv, mediaSequence+i); err != nil {
			return fmt.Errorf("failed to decrypt segment %s: %w", segmentPath, err)
		}
	}
	return nil
}

func decryptSegmentInPlace(
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
		return fmt.Errorf("segment file is empty")
	}
	if len(encryptedData)%aes.BlockSize != 0 {
		return fmt.Errorf("encrypted data length is not a multiple of block size")
	}
	iv := calculateSegmentIV(baseIV, segmentSequence)
	mode := cipher.NewCBCDecrypter(block, iv)
	decryptedData := make([]byte, len(encryptedData))
	mode.CryptBlocks(decryptedData, encryptedData)
	unpaddedData, err := removePKCS7Padding(decryptedData)
	if err != nil {
		return fmt.Errorf("failed to remove padding: %w", err)
	}

	if err := os.WriteFile(segmentPath, unpaddedData, 0644); err != nil {
		return fmt.Errorf("failed to write decrypted file: %w", err)
	}

	return nil
}

func DecryptSegmentStream(
	src io.Reader,
	dst io.Writer,
	key []byte,
	iv []byte,
	mediaSequence int,
) error {
	if !isValidAESKey(key) {
		return fmt.Errorf("invalid key: expected 16 bytes, got %d", len(key))
	}
	if !isValidIV(iv) {
		return fmt.Errorf("invalid IV: expected 16 bytes, got %d", len(iv))
	}
	encryptedData, err := io.ReadAll(src)
	if err != nil {
		return fmt.Errorf("failed to read encrypted data: %w", err)
	}
	if len(encryptedData) == 0 {
		return fmt.Errorf("no data to decrypt")
	}
	if len(encryptedData)%aes.BlockSize != 0 {
		return fmt.Errorf("encrypted data length is not a multiple of block size")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("failed to create AES cipher: %w", err)
	}
	segmentIV := calculateSegmentIV(iv, mediaSequence)
	mode := cipher.NewCBCDecrypter(block, segmentIV)
	decryptedData := make([]byte, len(encryptedData))
	mode.CryptBlocks(decryptedData, encryptedData)
	unpaddedData, err := removePKCS7Padding(decryptedData)
	if err != nil {
		return fmt.Errorf("failed to remove padding: %w", err)
	}
	if _, err := dst.Write(unpaddedData); err != nil {
		return fmt.Errorf("failed to write decrypted data: %w", err)
	}
	return nil
}

func DecryptSegmentBytes(
	encryptedData []byte,
	key []byte,
	iv []byte,
	mediaSequence int,
) ([]byte, error) {
	if !isValidAESKey(key) {
		return nil, fmt.Errorf("invalid key: expected 16 bytes, got %d", len(key))
	}
	if !isValidIV(iv) {
		return nil, fmt.Errorf("invalid IV: expected 16 bytes, got %d", len(iv))
	}

	if len(encryptedData) == 0 {
		return nil, fmt.Errorf("no data to decrypt")
	}
	if len(encryptedData)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("encrypted data length is not a multiple of block size")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}
	segmentIV := calculateSegmentIV(iv, mediaSequence)
	mode := cipher.NewCBCDecrypter(block, segmentIV)
	decryptedData := make([]byte, len(encryptedData))
	mode.CryptBlocks(decryptedData, encryptedData)
	unpaddedData, err := removePKCS7Padding(decryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to remove padding: %w", err)
	}

	return unpaddedData, nil
}

func DecryptSegmentsWithSequences(
	segments []string,
	key []byte,
	iv []byte,
	mediaSequences []int,
) error {
	if len(segments) != len(mediaSequences) {
		return fmt.Errorf("segments and mediaSequences arrays must have the same length")
	}

	if !isValidAESKey(key) {
		return fmt.Errorf("invalid key: expected 16 bytes, got %d", len(key))
	}
	if !isValidIV(iv) {
		return fmt.Errorf("invalid IV: expected 16 bytes, got %d", len(iv))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("failed to create AES cipher: %w", err)
	}

	for i, segmentPath := range segments {
		if err := decryptSegment(segmentPath, block, iv, mediaSequences[i]); err != nil {
			return fmt.Errorf("failed to decrypt segment %s (sequence %d): %w", segmentPath, mediaSequences[i], err)
		}
	}

	return nil
}

func DecryptSegmentsWithSequencesInPlace(
	segments []string,
	key []byte,
	iv []byte,
	mediaSequences []int,
) error {
	if len(segments) != len(mediaSequences) {
		return fmt.Errorf("segments and mediaSequences arrays must have the same length")
	}

	if !isValidAESKey(key) {
		return fmt.Errorf("invalid key: expected 16 bytes, got %d", len(key))
	}
	if !isValidIV(iv) {
		return fmt.Errorf("invalid IV: expected 16 bytes, got %d", len(iv))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("failed to create AES cipher: %w", err)
	}

	for i, segmentPath := range segments {
		if err := decryptSegmentInPlace(segmentPath, block, iv, mediaSequences[i]); err != nil {
			return fmt.Errorf("failed to decrypt segment %s (sequence %d): %w", segmentPath, mediaSequences[i], err)
		}
	}

	return nil
}

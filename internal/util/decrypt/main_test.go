package decrypt

import (
	"crypto/aes"
	"os"
	"path/filepath"
	"testing"
)

func TestIsValidAESKey(t *testing.T) {
	tests := []struct {
		name     string
		key      []byte
		expected bool
	}{
		{
			name:     "valid 16 byte key",
			key:      make([]byte, 16),
			expected: true,
		},
		{
			name:     "invalid short key",
			key:      make([]byte, 8),
			expected: false,
		},
		{
			name:     "invalid long key",
			key:      make([]byte, 20),
			expected: false,
		},
		{
			name:     "empty key",
			key:      []byte{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidAESKey(tt.key)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsValidIV(t *testing.T) {
	tests := []struct {
		name     string
		iv       []byte
		expected bool
	}{
		{
			name:     "valid 16 byte IV",
			iv:       make([]byte, 16),
			expected: true,
		},
		{
			name:     "invalid short IV",
			iv:       make([]byte, 8),
			expected: false,
		},
		{
			name:     "invalid long IV",
			iv:       make([]byte, 20),
			expected: false,
		},
		{
			name:     "empty IV",
			iv:       []byte{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidIV(tt.iv)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDecryptSegments(t *testing.T) {
	tests := []struct {
		name        string
		key         []byte
		iv          []byte
		expectError bool
	}{
		{
			name:        "invalid key size",
			key:         make([]byte, 8),
			iv:          make([]byte, 16),
			expectError: true,
		},
		{
			name:        "invalid IV size",
			key:         make([]byte, 16),
			iv:          make([]byte, 8),
			expectError: true,
		},
		{
			name:        "valid key and IV with no segments",
			key:         make([]byte, 16),
			iv:          make([]byte, 16),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments := []string{}
			err := DecryptSegments(segments, tt.key, tt.iv, 0)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestDecryptSegmentEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty_segment.ts")

	err := os.WriteFile(testFile, []byte{}, 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	key := make([]byte, 16)
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("failed to create cipher: %v", err)
	}

	iv := make([]byte, 16)
	err = decryptSegment(testFile, block, iv, 0)
	if err == nil {
		t.Error("expected error for empty file, got nil")
	}
}

func TestDecryptSegmentNonExistentFile(t *testing.T) {
	key := make([]byte, 16)
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("failed to create cipher: %v", err)
	}

	iv := make([]byte, 16)
	err = decryptSegment("/non/existent/file.ts", block, iv, 0)
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

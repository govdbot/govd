package decrypt

import (
	"testing"
)

func TestIsValidAESKeyUtil(t *testing.T) {
	validKey := make([]byte, 16)
	if !isValidAESKey(validKey) {
		t.Error("16 byte key should be valid")
	}

	invalidKey := make([]byte, 15)
	if isValidAESKey(invalidKey) {
		t.Error("15 byte key should be invalid")
	}
}

func TestIsValidIVUtil(t *testing.T) {
	validIV := make([]byte, 16)
	if !isValidIV(validIV) {
		t.Error("16 byte IV should be valid")
	}

	invalidIV := make([]byte, 15)
	if isValidIV(invalidIV) {
		t.Error("15 byte IV should be invalid")
	}
}

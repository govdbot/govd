package models

type DecryptionKey struct {
	Key           []byte // encoded key for AES decryption
	IV            []byte // initialization vector for AES decryption
	Method        string // e.g., "AES-128-CBC"
	MediaSequence int    // sequence number for HLS segments
}

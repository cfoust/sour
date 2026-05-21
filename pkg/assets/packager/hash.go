package packager

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

// HashFile computes the SHA-256 hash of a file's contents.
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// HashBytes computes the SHA-256 hash of raw bytes.
func HashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

// HashString computes the SHA-256 hash of a UTF-8 string.
func HashString(s string) string {
	return HashBytes([]byte(s))
}

package utils

import (
	"crypto/sha256"
	"fmt"
)

func Hash(data []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func HashString(value string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(value)))
}

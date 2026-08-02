package domain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func NewID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

func ContentID(algorithm, digestHex string, size int64) string {
	return fmt.Sprintf("%s:%s:%d", algorithm, digestHex, size)
}

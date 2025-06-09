package utils

import (
	"crypto/rand"
	"math/big"
)

const (
	// Character set for short codes (alphanumeric, case-sensitive)
	charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	// Length of the short code
	shortCodeLength = 6
)

// GenerateShortCode generates a random short code for URL shortening
func GenerateShortCode() string {
	shortCode := make([]byte, shortCodeLength)
	charsetLength := big.NewInt(int64(len(charset)))

	for i := range shortCode {
		randomIndex, _ := rand.Int(rand.Reader, charsetLength)
		shortCode[i] = charset[randomIndex.Int64()]
	}

	return string(shortCode)
}

package utils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// ValidateSessionHash checks if the session hash is valid for a user
func ValidateSessionHash(userID, challengeID, password, providedHash, sessionHashKey string) bool {
	expected := GenerateSessionHash(userID, challengeID, password, sessionHashKey)
	return hmac.Equal([]byte(expected), []byte(providedHash))
}

// GenerateSessionHash generates a HMAC SHA-256 session key
func GenerateSessionHash(userID, challengeID, password, sessionHashKey string) string {
	data := userID + challengeID + password
	h := hmac.New(sha256.New, []byte(sessionHashKey))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func GenerateBigCapPassword(length int) string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return ""
	}

	for i := 0; i < length; i++ {
		bytes[i] = letters[int(bytes[i])%len(letters)]
	}
	return string(bytes)
}

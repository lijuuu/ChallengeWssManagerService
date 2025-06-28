package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"sync"

	"github.com/lijuuu/ChallengeWssManagerService/internal/models"
)

var (
	activeSessions = make(map[string]*models.Session)
	sessionMu      sync.RWMutex
)

// ValidateSessionHash checks if the session hash is valid for a user
func ValidateSessionHash(userID, challengeID, password, providedHash string) bool {
	expected := GenerateSessionHash(userID, challengeID, password)
	return hmac.Equal([]byte(expected), []byte(providedHash))
}

// GenerateSessionHash generates a HMAC SHA-256 session key
func GenerateSessionHash(userID, challengeID, password string) string {
	data := userID + challengeID + password
	h := hmac.New(sha256.New, []byte(models.SessionHashKey))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func SetSession(key string, session *models.Session) {
	sessionMu.Lock()
	activeSessions[key] = session
	sessionMu.Unlock()
}

func DeleteSession(key string) {
	sessionMu.Lock()
	delete(activeSessions, key)
	sessionMu.Unlock()
}

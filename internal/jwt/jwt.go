package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// customclaims defines the claims we store in jwts for this service.
// it includes who the user is and which challenge they belong to.
type CustomClaims struct {
	UserID      string `json:"userId"`
	ChallengeID string `json:"challengeId"`
	EntityType  string `json:"entityType"`
	jwt.RegisteredClaims
}

// jwtmanager signs and verifies jwt tokens using an hmac secret.
type JWTManager struct {
	secretKey []byte
}

// newjwtmanager returns a manager that uses the provided secret key.
func NewJWTManager(secretKey string) *JWTManager {
	return &JWTManager{
		secretKey: []byte(secretKey),
	}
}

// generatetoken issues a jwt for a user in a given challenge.
// the token expires after the provided duration.
func (j *JWTManager) GenerateToken(userID, challengeID string, expiration time.Duration) (string, error) {
	if userID == "" {
		return "", errors.New("userID cannot be empty")
	}
	if challengeID == "" {
		return "", errors.New("challengeID cannot be empty")
	}

	claims := CustomClaims{
		UserID:      userID,
		ChallengeID: challengeID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

// validatetoken parses and verifies a jwt and returns its claims if valid.
func (j *JWTManager) ValidateToken(tokenString string) (*CustomClaims, error) {
	if tokenString == "" {
		return nil, errors.New("token cannot be empty")
	}

	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		//only accept hmac signatures.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return j.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// extractclaims reads claims without verifying the signature.
// useful for lightweight checks in middleware. do not use for auth decisions.
func (j *JWTManager) ExtractClaims(tokenString string) (*CustomClaims, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &CustomClaims{})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok {
		return claims, nil
	}

	return nil, errors.New("invalid token claims")
}

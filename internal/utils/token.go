package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

// Claims are the access-token claims. SessionID ties the token to the
// refresh-token session stored in Redis.
type Claims struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

func accessSecret() ([]byte, error) {
	secret := os.Getenv("JWT_ACCESS_SECRET")
	if secret == "" {
		return nil, errors.New("JWT_ACCESS_SECRET is not set")
	}
	return []byte(secret), nil
}

func durationFromEnv(env string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(env)
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration like 15m or 168h", env)
	}
	return d, nil
}

// AccessTTL is how long an access token (and its cookie) lives.
func AccessTTL() (time.Duration, error) {
	return durationFromEnv("JWT_ACCESS_TTL", 15*time.Minute)
}

// RefreshTTL is how long a refresh token (and its Redis session) lives.
func RefreshTTL() (time.Duration, error) {
	return durationFromEnv("JWT_REFRESH_TTL", 7*24*time.Hour)
}

// GenerateAccessToken signs a short-lived JWT; it is verified without any Redis lookup.
func GenerateAccessToken(userID, sessionID string) (string, error) {
	secret, err := accessSecret()
	if err != nil {
		return "", err
	}
	ttl, err := AccessTTL()
	if err != nil {
		return "", err
	}

	jti, err := randomBytes(16)
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := Claims{
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        hex.EncodeToString(jti),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

// ParseAccessToken verifies signature and expiry.
func ParseAccessToken(tokenString string) (*Claims, error) {
	secret, err := accessSecret()
	if err != nil {
		return nil, err
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(*jwt.Token) (any, error) {
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// NewOpaqueToken returns a random 256-bit URL-safe token (used for refresh tokens and session IDs).
func NewOpaqueToken() (string, error) {
	b, err := randomBytes(32)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashToken is what gets stored in Redis instead of the raw token.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

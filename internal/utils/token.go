package utils

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	Type TokenType `json:"type"`
	jwt.RegisteredClaims
}

func secretFor(typ TokenType) ([]byte, error) {
	env := "JWT_ACCESS_SECRET"
	if typ == RefreshToken {
		env = "JWT_REFRESH_SECRET"
	}
	secret := os.Getenv(env)
	if secret == "" {
		return nil, fmt.Errorf("%s is not set", env)
	}
	return []byte(secret), nil
}

// ttlFor reads the token lifetime (e.g. "15m", "168h") from the environment,
// falling back to a default when unset.
func ttlFor(typ TokenType) (time.Duration, error) {
	env, def := "JWT_ACCESS_TTL", time.Hour
	if typ == RefreshToken {
		env, def = "JWT_REFRESH_TTL", 7*24*time.Hour
	}
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

func generateToken(userID string, typ TokenType) (string, error) {
	secret, err := secretFor(typ)
	if err != nil {
		return "", err
	}
	ttl, err := ttlFor(typ)
	if err != nil {
		return "", err
	}

	jti := make([]byte, 16)
	if _, err := rand.Read(jti); err != nil {
		return "", err
	}

	now := time.Now()
	claims := Claims{
		Type: typ,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        hex.EncodeToString(jti),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

func GenerateAccessToken(userID string) (string, error) {
	return generateToken(userID, AccessToken)
}

func GenerateRefreshToken(userID string) (string, error) {
	return generateToken(userID, RefreshToken)
}

// ParseToken verifies signature, expiry and that the token is of type typ.
func ParseToken(tokenString string, typ TokenType) (*Claims, error) {
	secret, err := secretFor(typ)
	if err != nil {
		return nil, err
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(*jwt.Token) (any, error) {
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil || !token.Valid || claims.Type != typ {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func ValidateRefreshToken(tokenString string) (*Claims, error) {
	claims, err := ParseToken(tokenString, RefreshToken)
	if err != nil {
		return nil, err
	}
	return claims, nil
}

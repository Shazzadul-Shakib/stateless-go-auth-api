package auth

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Shazzadul-Shakib/go-auth-api/internal/utils"
	"github.com/redis/go-redis/v9"
)

const (
	refreshPrefix      = "refresh:"       // refresh:{hash}      -> session record
	usedPrefix         = "used:"          // used:{hash}         -> record of an already-rotated token (reuse detection)
	sessionPrefix      = "session:"       // session:{sid}       -> hash of the session's current refresh token
	userSessionsPrefix = "user_sessions:" // user_sessions:{uid} -> set of session IDs
)

type sessionRecord struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
}

// SessionStore keeps refresh-token sessions in Redis so the API servers stay stateless.
type SessionStore struct{ rdb *redis.Client }

func NewSessionStore(rdb *redis.Client) *SessionStore {
	return &SessionStore{rdb: rdb}
}

// Create starts a new session and returns its ID and first refresh token.
func (s *SessionStore) Create(ctx context.Context, userID string) (sessionID, refreshToken string, err error) {
	sessionID, err = utils.NewOpaqueToken()
	if err != nil {
		return "", "", err
	}
	refreshToken, err = s.save(ctx, sessionRecord{UserID: userID, SessionID: sessionID}, "")
	return sessionID, refreshToken, err
}

// Rotate consumes a refresh token (single use) and issues the next one for the same session.
// Presenting an already-rotated token is treated as theft: the whole session is revoked.
func (s *SessionStore) Rotate(ctx context.Context, refreshToken string) (rec sessionRecord, newToken string, err error) {
	hash := utils.HashToken(refreshToken)

	raw, err := s.rdb.GetDel(ctx, refreshPrefix+hash).Result()
	if errors.Is(err, redis.Nil) {
		if used, uerr := s.rdb.Get(ctx, usedPrefix+hash).Result(); uerr == nil {
			var old sessionRecord
			if json.Unmarshal([]byte(used), &old) == nil {
				_ = s.revokeSession(ctx, old)
			}
		}
		return rec, "", ErrInvalidToken
	}
	if err != nil {
		return rec, "", err
	}
	if err := json.Unmarshal([]byte(raw), &rec); err != nil {
		return rec, "", err
	}

	newToken, err = s.save(ctx, rec, hash)
	return rec, newToken, err
}

// Revoke ends the session that owns refreshToken. Unknown tokens are ignored (idempotent logout).
func (s *SessionStore) Revoke(ctx context.Context, refreshToken string) error {
	raw, err := s.rdb.Get(ctx, refreshPrefix+utils.HashToken(refreshToken)).Result()
	if errors.Is(err, redis.Nil) {
		return nil
	}
	if err != nil {
		return err
	}
	var rec sessionRecord
	if err := json.Unmarshal([]byte(raw), &rec); err != nil {
		return err
	}
	return s.revokeSession(ctx, rec)
}

// RevokeAll ends every session of a user.
func (s *SessionStore) RevokeAll(ctx context.Context, userID string) error {
	sids, err := s.rdb.SMembers(ctx, userSessionsPrefix+userID).Result()
	if err != nil {
		return err
	}
	for _, sid := range sids {
		if err := s.revokeSession(ctx, sessionRecord{UserID: userID, SessionID: sid}); err != nil {
			return err
		}
	}
	return s.rdb.Del(ctx, userSessionsPrefix+userID).Err()
}

// save stores a fresh refresh token for rec. oldHash, when set, is the token being replaced.
func (s *SessionStore) save(ctx context.Context, rec sessionRecord, oldHash string) (string, error) {
	ttl, err := utils.RefreshTTL()
	if err != nil {
		return "", err
	}
	token, err := utils.NewOpaqueToken()
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return "", err
	}
	hash := utils.HashToken(token)

	pipe := s.rdb.TxPipeline()
	pipe.Set(ctx, refreshPrefix+hash, data, ttl)
	pipe.Set(ctx, sessionPrefix+rec.SessionID, hash, ttl)
	pipe.SAdd(ctx, userSessionsPrefix+rec.UserID, rec.SessionID)
	pipe.Expire(ctx, userSessionsPrefix+rec.UserID, ttl)
	if oldHash != "" {
		pipe.Set(ctx, usedPrefix+oldHash, data, ttl)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return "", err
	}
	return token, nil
}

func (s *SessionStore) revokeSession(ctx context.Context, rec sessionRecord) error {
	hash, err := s.rdb.Get(ctx, sessionPrefix+rec.SessionID).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}
	pipe := s.rdb.TxPipeline()
	if hash != "" {
		pipe.Del(ctx, refreshPrefix+hash)
	}
	pipe.Del(ctx, sessionPrefix+rec.SessionID)
	pipe.SRem(ctx, userSessionsPrefix+rec.UserID, rec.SessionID)
	_, err = pipe.Exec(ctx)
	return err
}

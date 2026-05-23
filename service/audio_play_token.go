package service

import (
	"errors"
	"strings"
	"sync"
	"time"

	"sealchat/utils"
)

const defaultAudioPlayTokenTTL = 15 * time.Minute

var (
	ErrAudioPlayTokenNotFound      = errors.New("audio play token not found")
	ErrAudioPlayTokenExpired       = errors.New("audio play token expired")
	ErrAudioPlayTokenAssetMismatch = errors.New("audio play token asset mismatch")
)

type AudioPlayTokenGrant struct {
	Token     string
	ExpiresAt time.Time
}

type AudioPlayTokenClaims struct {
	Token     string
	AssetID   string
	UserID    string
	ExpiresAt time.Time
}

type audioPlayTokenStore struct {
	mu    sync.Mutex
	items map[string]AudioPlayTokenClaims
}

var globalAudioPlayTokenStore = audioPlayTokenStore{
	items: map[string]AudioPlayTokenClaims{},
}

func IssueAudioPlayToken(userID, assetID string) (*AudioPlayTokenGrant, error) {
	return issueAudioPlayToken(userID, assetID, time.Now(), defaultAudioPlayTokenTTL)
}

func issueAudioPlayToken(userID, assetID string, now time.Time, ttl time.Duration) (*AudioPlayTokenGrant, error) {
	normalizedUserID := strings.TrimSpace(userID)
	normalizedAssetID := strings.TrimSpace(assetID)
	if normalizedUserID == "" {
		return nil, errors.New("user id is empty")
	}
	if normalizedAssetID == "" {
		return nil, errors.New("asset id is empty")
	}
	if ttl <= 0 {
		ttl = defaultAudioPlayTokenTTL
	}

	token := utils.NewID() + utils.NewIDWithLength(12)
	expiresAt := now.Add(ttl)
	claims := AudioPlayTokenClaims{
		Token:     token,
		AssetID:   normalizedAssetID,
		UserID:    normalizedUserID,
		ExpiresAt: expiresAt,
	}

	globalAudioPlayTokenStore.mu.Lock()
	defer globalAudioPlayTokenStore.mu.Unlock()
	globalAudioPlayTokenStore.cleanupExpiredLocked(now)
	globalAudioPlayTokenStore.items[token] = claims

	return &AudioPlayTokenGrant{
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

func ResolveAudioPlayToken(token, assetID string) (*AudioPlayTokenClaims, error) {
	normalizedToken := strings.TrimSpace(token)
	normalizedAssetID := strings.TrimSpace(assetID)
	if normalizedToken == "" {
		return nil, ErrAudioPlayTokenNotFound
	}
	if normalizedAssetID == "" {
		return nil, ErrAudioPlayTokenAssetMismatch
	}

	now := time.Now()
	globalAudioPlayTokenStore.mu.Lock()
	defer globalAudioPlayTokenStore.mu.Unlock()

	claims, ok := globalAudioPlayTokenStore.items[normalizedToken]
	if !ok {
		globalAudioPlayTokenStore.cleanupExpiredLocked(now)
		return nil, ErrAudioPlayTokenNotFound
	}
	if !claims.ExpiresAt.After(now) {
		delete(globalAudioPlayTokenStore.items, normalizedToken)
		return nil, ErrAudioPlayTokenExpired
	}
	if claims.AssetID != normalizedAssetID {
		return nil, ErrAudioPlayTokenAssetMismatch
	}
	result := claims
	return &result, nil
}

func resetAudioPlayTokenStore() {
	globalAudioPlayTokenStore.mu.Lock()
	defer globalAudioPlayTokenStore.mu.Unlock()
	globalAudioPlayTokenStore.items = map[string]AudioPlayTokenClaims{}
}

func ResetAudioPlayTokenStoreForTests() {
	resetAudioPlayTokenStore()
}

func (s *audioPlayTokenStore) cleanupExpiredLocked(now time.Time) {
	for token, claims := range s.items {
		if !claims.ExpiresAt.After(now) {
			delete(s.items, token)
		}
	}
}

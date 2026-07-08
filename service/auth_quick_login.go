package service

import (
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/blake2s"

	"sealchat/model"
	"sealchat/utils"
)

const DefaultQuickLoginTTL = time.Minute
const quickLoginTerminalRetention = 10 * time.Minute

var (
	ErrQuickLoginRequestNotFound              = errors.New("quick login request not found")
	ErrQuickLoginRequestNotPending            = errors.New("quick login request not pending")
	ErrQuickLoginRequesterTokenMismatch       = errors.New("quick login requester token mismatch")
	ErrQuickLoginRequesterFingerprintMismatch = errors.New("quick login requester fingerprint mismatch")
)

type QuickLoginStatus string

const (
	QuickLoginStatusPending  QuickLoginStatus = "pending"
	QuickLoginStatusApproved QuickLoginStatus = "approved"
	QuickLoginStatusDenied   QuickLoginStatus = "denied"
	QuickLoginStatusExpired  QuickLoginStatus = "expired"
	QuickLoginStatusConsumed QuickLoginStatus = "consumed"
)

type QuickLoginGrant struct {
	RequestID      string
	RequesterToken string
	ExpiresAt      time.Time
}

type QuickLoginRequest struct {
	RequestID              string
	RequesterToken         string
	TargetUserID           string
	AccountInput           string
	RequesterBrowser       string
	RequesterDevice        string
	RequesterIP            string
	RequesterUA            string
	RequesterFingerprint   string
	Status                 QuickLoginStatus
	AvailableApprover      int
	ApprovedByUserID       string
	ApprovedByConnectionID string
	CreatedAt              time.Time
	ExpiresAt              time.Time
	ApprovedAt             *time.Time
	DeniedAt               *time.Time
	ConsumedAt             *time.Time
	IssuedToken            string
}

type CreateQuickLoginRequestInput struct {
	TargetUserID      string
	AccountInput      string
	RequesterIP       string
	RequesterUA       string
	RequesterBrowser  string
	RequesterDevice   string
	AvailableApprover int
}

type ApproveQuickLoginRequestInput struct {
	RequestID    string
	ApproverID   string
	ConnectionID string
}

type DenyQuickLoginRequestInput struct {
	RequestID string
}

type PollQuickLoginRequestInput struct {
	RequestID      string
	RequesterToken string
	RequesterIP    string
	RequesterUA    string
}

type ConsumeQuickLoginRequestInput struct {
	RequestID      string
	RequesterToken string
	RequesterIP    string
	RequesterUA    string
}

type QuickLoginPollResult struct {
	Status  QuickLoginStatus
	Request QuickLoginRequest
	Token   string
}

type quickLoginStore struct {
	mu    sync.Mutex
	items map[string]QuickLoginRequest
}

var globalQuickLoginStore = quickLoginStore{
	items: map[string]QuickLoginRequest{},
}

func buildQuickLoginFingerprint(ip, userAgent string) string {
	sum := blake2s.Sum256([]byte(strings.TrimSpace(ip) + "\n" + strings.TrimSpace(userAgent)))
	return base64.RawStdEncoding.EncodeToString(sum[:])
}

func CreateQuickLoginRequest(input CreateQuickLoginRequestInput, now time.Time, ttl time.Duration) (*QuickLoginGrant, error) {
	targetUserID := strings.TrimSpace(input.TargetUserID)
	if targetUserID == "" {
		return nil, errors.New("target user is empty")
	}
	if ttl <= 0 {
		ttl = DefaultQuickLoginTTL
	}
	record := QuickLoginRequest{
		RequestID:            utils.NewID(),
		RequesterToken:       utils.NewID() + utils.NewIDWithLength(12),
		TargetUserID:         targetUserID,
		AccountInput:         strings.TrimSpace(input.AccountInput),
		RequesterBrowser:     strings.TrimSpace(input.RequesterBrowser),
		RequesterDevice:      strings.TrimSpace(input.RequesterDevice),
		RequesterIP:          strings.TrimSpace(input.RequesterIP),
		RequesterUA:          strings.TrimSpace(input.RequesterUA),
		RequesterFingerprint: buildQuickLoginFingerprint(input.RequesterIP, input.RequesterUA),
		Status:               QuickLoginStatusPending,
		AvailableApprover:    input.AvailableApprover,
		CreatedAt:            now,
		ExpiresAt:            now.Add(ttl),
	}

	globalQuickLoginStore.mu.Lock()
	defer globalQuickLoginStore.mu.Unlock()
	globalQuickLoginStore.cleanupExpiredLocked(now)
	globalQuickLoginStore.items[record.RequestID] = record

	return &QuickLoginGrant{
		RequestID:      record.RequestID,
		RequesterToken: record.RequesterToken,
		ExpiresAt:      record.ExpiresAt,
	}, nil
}

func ApproveQuickLoginRequest(input ApproveQuickLoginRequestInput, now time.Time) (*QuickLoginRequest, error) {
	globalQuickLoginStore.mu.Lock()
	defer globalQuickLoginStore.mu.Unlock()

	record, err := globalQuickLoginStore.getLocked(strings.TrimSpace(input.RequestID), now)
	if err != nil {
		return nil, err
	}
	if record.Status != QuickLoginStatusPending {
		return nil, ErrQuickLoginRequestNotPending
	}
	record.Status = QuickLoginStatusApproved
	record.ApprovedByUserID = strings.TrimSpace(input.ApproverID)
	record.ApprovedByConnectionID = strings.TrimSpace(input.ConnectionID)
	record.ApprovedAt = &now
	globalQuickLoginStore.items[record.RequestID] = record
	clone := record
	return &clone, nil
}

func DenyQuickLoginRequest(input DenyQuickLoginRequestInput, now time.Time) (*QuickLoginRequest, error) {
	globalQuickLoginStore.mu.Lock()
	defer globalQuickLoginStore.mu.Unlock()

	record, err := globalQuickLoginStore.getLocked(strings.TrimSpace(input.RequestID), now)
	if err != nil {
		return nil, err
	}
	if record.Status != QuickLoginStatusPending {
		return nil, ErrQuickLoginRequestNotPending
	}
	record.Status = QuickLoginStatusDenied
	record.DeniedAt = &now
	globalQuickLoginStore.items[record.RequestID] = record
	clone := record
	return &clone, nil
}

func PollQuickLoginRequest(input PollQuickLoginRequestInput, now time.Time) (*QuickLoginPollResult, error) {
	globalQuickLoginStore.mu.Lock()
	defer globalQuickLoginStore.mu.Unlock()

	record, err := globalQuickLoginStore.getLocked(strings.TrimSpace(input.RequestID), now)
	if err != nil {
		return nil, err
	}
	if err := validateRequesterLocked(record, strings.TrimSpace(input.RequesterToken), input.RequesterIP, input.RequesterUA); err != nil {
		return nil, err
	}
	return &QuickLoginPollResult{Status: record.Status, Request: record, Token: record.IssuedToken}, nil
}

func ConsumeQuickLoginRequest(input ConsumeQuickLoginRequestInput, now time.Time) (*QuickLoginPollResult, error) {
	globalQuickLoginStore.mu.Lock()
	defer globalQuickLoginStore.mu.Unlock()

	record, err := globalQuickLoginStore.getLocked(strings.TrimSpace(input.RequestID), now)
	if err != nil {
		return nil, err
	}
	if err := validateRequesterLocked(record, strings.TrimSpace(input.RequesterToken), input.RequesterIP, input.RequesterUA); err != nil {
		return nil, err
	}
	if record.Status == QuickLoginStatusApproved {
		if strings.TrimSpace(record.IssuedToken) == "" {
			token, err := model.UserGenerateAccessToken(record.TargetUserID)
			if err != nil {
				return nil, err
			}
			record.IssuedToken = token
		}
		record.Status = QuickLoginStatusConsumed
		record.ConsumedAt = &now
		globalQuickLoginStore.items[record.RequestID] = record
	}
	return &QuickLoginPollResult{Status: record.Status, Request: record, Token: record.IssuedToken}, nil
}

func ResetQuickLoginStoreForTests() {
	globalQuickLoginStore.mu.Lock()
	defer globalQuickLoginStore.mu.Unlock()
	globalQuickLoginStore.items = map[string]QuickLoginRequest{}
}

func GetQuickLoginRequest(requestID string, now time.Time) (*QuickLoginRequest, error) {
	globalQuickLoginStore.mu.Lock()
	defer globalQuickLoginStore.mu.Unlock()

	record, err := globalQuickLoginStore.getLocked(strings.TrimSpace(requestID), now)
	if err != nil {
		return nil, err
	}
	clone := record
	return &clone, nil
}

func validateRequesterLocked(record QuickLoginRequest, requesterToken, requesterIP, requesterUA string) error {
	if record.RequesterToken != requesterToken {
		return ErrQuickLoginRequesterTokenMismatch
	}
	if record.RequesterFingerprint != buildQuickLoginFingerprint(requesterIP, requesterUA) {
		return ErrQuickLoginRequesterFingerprintMismatch
	}
	return nil
}

func (s *quickLoginStore) getLocked(requestID string, now time.Time) (QuickLoginRequest, error) {
	if requestID == "" {
		return QuickLoginRequest{}, ErrQuickLoginRequestNotFound
	}
	s.cleanupExpiredLocked(now)
	record, ok := s.items[requestID]
	if !ok {
		return QuickLoginRequest{}, ErrQuickLoginRequestNotFound
	}
	return record, nil
}

func (s *quickLoginStore) cleanupExpiredLocked(now time.Time) {
	for requestID, record := range s.items {
		if record.Status == QuickLoginStatusPending && !record.ExpiresAt.After(now) {
			record.Status = QuickLoginStatusExpired
			s.items[requestID] = record
		}
		terminalAt := quickLoginTerminalTime(record)
		if !terminalAt.IsZero() && now.Sub(terminalAt) >= quickLoginTerminalRetention {
			delete(s.items, requestID)
		}
	}
}

func quickLoginTerminalTime(record QuickLoginRequest) time.Time {
	switch record.Status {
	case QuickLoginStatusConsumed:
		if record.ConsumedAt != nil {
			return *record.ConsumedAt
		}
	case QuickLoginStatusDenied:
		if record.DeniedAt != nil {
			return *record.DeniedAt
		}
		if !record.ExpiresAt.IsZero() {
			return record.ExpiresAt
		}
		return record.CreatedAt
	case QuickLoginStatusExpired:
		return record.ExpiresAt
	}
	return time.Time{}
}

package api

import (
	"strings"
	"sync"
	"time"

	"sealchat/utils"
)

type quickLoginRequestRateState struct {
	windowStart time.Time
	count       int
}

type quickLoginRequestRateLimiter struct {
	mu          sync.Mutex
	entries     map[string]quickLoginRequestRateState
	nextCleanup time.Time
}

var globalQuickLoginRequestRateLimiter = quickLoginRequestRateLimiter{
	entries: map[string]quickLoginRequestRateState{},
}

func normalizeQuickLoginRateKey(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return "unknown"
	}
	return trimmed
}

func resolveQuickLoginRequestRateLimitConfig(cfg *utils.AppConfig) utils.QuickLoginRequestRateLimitConfig {
	if cfg == nil {
		return utils.QuickLoginRequestRateLimitConfig{
			WindowSeconds:    60,
			MaxPerIP:         6,
			MaxPerTargetUser: 3,
			MaxPerAccount:    3,
		}
	}
	return cfg.QuickLogin.RequestRateLimit
}

func allowQuickLoginRequest(clientIP, targetUserID, accountInput string, now time.Time, cfg *utils.AppConfig) (bool, time.Duration) {
	limitCfg := resolveQuickLoginRequestRateLimitConfig(cfg)
	windowSeconds := limitCfg.WindowSeconds
	if windowSeconds <= 0 {
		return true, 0
	}
	window := time.Duration(windowSeconds) * time.Second
	clientIP = normalizeQuickLoginRateKey(clientIP)
	targetUserID = normalizeQuickLoginRateKey(targetUserID)
	accountInput = normalizeQuickLoginRateKey(accountInput)

	limiter := &globalQuickLoginRequestRateLimiter
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	if now.After(limiter.nextCleanup) {
		for key, state := range limiter.entries {
			if now.Sub(state.windowStart) >= window {
				delete(limiter.entries, key)
			}
		}
		limiter.nextCleanup = now.Add(window)
	}

	if allowed, retryAfter := consumeQuickLoginRequestQuotaLocked("ip|"+clientIP, limitCfg.MaxPerIP, window, now); !allowed {
		return false, retryAfter
	}
	if allowed, retryAfter := consumeQuickLoginRequestQuotaLocked("user|"+targetUserID, limitCfg.MaxPerTargetUser, window, now); !allowed {
		return false, retryAfter
	}
	return consumeQuickLoginRequestQuotaLocked("account|"+accountInput, limitCfg.MaxPerAccount, window, now)
}

func consumeQuickLoginRequestQuotaLocked(key string, limit int, window time.Duration, now time.Time) (bool, time.Duration) {
	if limit <= 0 {
		return true, 0
	}
	entry := globalQuickLoginRequestRateLimiter.entries[key]
	if entry.windowStart.IsZero() || now.Sub(entry.windowStart) >= window {
		globalQuickLoginRequestRateLimiter.entries[key] = quickLoginRequestRateState{
			windowStart: now,
			count:       1,
		}
		return true, 0
	}
	if entry.count >= limit {
		retryAfter := window - now.Sub(entry.windowStart)
		if retryAfter < time.Second {
			retryAfter = time.Second
		}
		return false, retryAfter
	}
	entry.count++
	globalQuickLoginRequestRateLimiter.entries[key] = entry
	return true, 0
}

func resetQuickLoginRequestRateLimiterForTests() {
	globalQuickLoginRequestRateLimiter.mu.Lock()
	defer globalQuickLoginRequestRateLimiter.mu.Unlock()
	globalQuickLoginRequestRateLimiter.entries = map[string]quickLoginRequestRateState{}
	globalQuickLoginRequestRateLimiter.nextCleanup = time.Time{}
}

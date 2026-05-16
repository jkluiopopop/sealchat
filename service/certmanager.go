package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/mholt/acmez/v3/acme"
	"go.uber.org/zap"

	"sealchat/utils"
)

const (
	letEncryptIssuerEvent        = "certmagic"
	letsEncryptShortLivedProfile = "shortlived"
	certificateLogLimit          = 200
)

type CertificateManagerOptions struct {
	SkipObtain bool
}

type CertificateStatus struct {
	Enabled              bool       `json:"enabled"`
	RuntimeActive        bool       `json:"runtimeActive"`
	SubjectIP            string     `json:"subjectIp"`
	Issuer               string     `json:"issuer"`
	Challenge            string     `json:"challenge"`
	CertificatePresent   bool       `json:"certificatePresent"`
	NotBefore            *time.Time `json:"notBefore,omitempty"`
	NotAfter             *time.Time `json:"notAfter,omitempty"`
	RemainingDays        int        `json:"remainingDays"`
	LastError            string     `json:"lastError,omitempty"`
	LastCheckAt          *time.Time `json:"lastCheckAt,omitempty"`
	LastSuccessAt        *time.Time `json:"lastSuccessAt,omitempty"`
	NextCheckAt          *time.Time `json:"nextCheckAt,omitempty"`
	RetryCount           int        `json:"retryCount"`
	Retrying             bool       `json:"retrying"`
	RenewBeforeDays      int        `json:"renewBeforeDays"`
	CheckIntervalMinutes int        `json:"checkIntervalMinutes"`
	RetryInitialMinutes  int        `json:"retryInitialMinutes"`
	RetryMaxMinutes      int        `json:"retryMaxMinutes"`
}

type CertificateLogEntry struct {
	Time      time.Time `json:"time"`
	Level     string    `json:"level"`
	Event     string    `json:"event"`
	Message   string    `json:"message"`
	SubjectIP string    `json:"subjectIp,omitempty"`
	Issuer    string    `json:"issuer,omitempty"`
	Challenge string    `json:"challenge,omitempty"`
}

type CertificateManager struct {
	enabled    bool
	certConfig *certmagic.Config
	cache      *certmagic.Cache
	acmeIssuer *certmagic.ACMEIssuer
	zeroIssuer *certmagic.ZeroSSLIssuer
	subjectIP  string
	issuer     utils.CertificateIssuer
	challenge  utils.CertificateChallenge
	lastError  string
	logs       []CertificateLogEntry
	logsMu     sync.Mutex
	stateMu    sync.Mutex
	obtainMu   sync.Mutex

	lastCheckAt          *time.Time
	lastSuccessAt        *time.Time
	nextCheckAt          *time.Time
	retryCount           int
	retrying             bool
	renewBeforeDays      int
	checkIntervalMinutes int
	retryInitialMinutes  int
	retryMaxMinutes      int
	renewalLoopCancel    context.CancelFunc
	renewalLoopRunner    func(context.Context) time.Duration
}

func NewCertificateManager(ctx context.Context, appCfg *utils.AppConfig) (*CertificateManager, error) {
	return NewCertificateManagerWithOptions(ctx, appCfg, CertificateManagerOptions{})
}

func NewCertificateManagerWithOptions(ctx context.Context, appCfg *utils.AppConfig, opts CertificateManagerOptions) (*CertificateManager, error) {
	manager := &CertificateManager{}
	if appCfg == nil || !appCfg.Certificate.Enabled {
		return manager, nil
	}

	cfg := utils.NormalizeCertificateConfig(appCfg.Certificate)
	if err := utils.ValidateCertificateConfig(cfg); err != nil {
		return nil, err
	}

	logger := zap.NewNop()
	storage := &certmagic.FileStorage{Path: cfg.StorageDir}
	var cache *certmagic.Cache
	cache = certmagic.NewCache(certmagic.CacheOptions{
		GetConfigForCert: func(cert certmagic.Certificate) (*certmagic.Config, error) {
			return certmagic.New(cache, certmagic.Config{
				Storage: storage,
				Logger:  logger,
			}), nil
		},
		Logger: logger,
	})

	manager.enabled = true
	manager.cache = cache
	manager.subjectIP = cfg.SubjectIP
	manager.issuer = cfg.Issuer
	manager.challenge = cfg.Challenge
	manager.renewBeforeDays = cfg.RenewBeforeDays
	manager.checkIntervalMinutes = cfg.CheckIntervalMinutes
	manager.retryInitialMinutes = cfg.RetryInitialMinutes
	manager.retryMaxMinutes = cfg.RetryMaxMinutes

	cmCfg := certmagic.New(cache, certmagic.Config{
		Storage:           storage,
		Logger:            logger,
		DefaultServerName: cfg.SubjectIP,
		OnEvent: func(ctx context.Context, event string, data map[string]any) error {
			manager.addLog("info", event, fmt.Sprintf("%s: %v", event, data))
			return nil
		},
	})
	issuer, acmeIssuer, zeroIssuer := manager.buildIssuer(cmCfg, storage, cfg, logger)
	cmCfg.Issuers = []certmagic.Issuer{issuer}
	manager.certConfig = cmCfg
	manager.acmeIssuer = acmeIssuer
	manager.zeroIssuer = zeroIssuer

	manager.addLog("info", letEncryptIssuerEvent, "证书管理器已初始化")
	if !opts.SkipObtain {
		if err := cmCfg.ManageSync(ctx, []string{cfg.SubjectIP}); err != nil {
			manager.lastError = err.Error()
			manager.addLog("error", "obtain", err.Error())
			return nil, err
		}
		manager.addLog("info", "obtain", "证书已进入 CertMagic 管理")
	}
	return manager, nil
}

func (m *CertificateManager) buildIssuer(cmCfg *certmagic.Config, storage certmagic.Storage, cfg utils.CertificateConfig, logger *zap.Logger) (certmagic.Issuer, *certmagic.ACMEIssuer, *certmagic.ZeroSSLIssuer) {
	if cfg.Issuer == utils.CertificateIssuerZeroSSL90Days && cfg.ZeroSSLAPIKey != "" && cfg.Challenge == utils.CertificateChallengeHTTP01 {
		issuer := &certmagic.ZeroSSLIssuer{
			APIKey:       cfg.ZeroSSLAPIKey,
			Storage:      storage,
			ValidityDays: 90,
			Logger:       logger,
		}
		return issuer, nil, issuer
	}

	template := certmagic.ACMEIssuer{
		Email:  cfg.Email,
		Agreed: true,
		Logger: logger,
	}
	switch cfg.Issuer {
	case utils.CertificateIssuerZeroSSL90Days:
		template.CA = certmagic.ZeroSSLProductionCA
		template.ExternalAccount = &acme.EAB{
			KeyID:  cfg.ZeroSSLEABKeyID,
			MACKey: cfg.ZeroSSLEABMACKey,
		}
	default:
		if cfg.Staging {
			template.CA = certmagic.LetsEncryptStagingCA
		} else {
			template.CA = certmagic.LetsEncryptProductionCA
		}
		template.TestCA = certmagic.LetsEncryptStagingCA
		template.Profile = letsEncryptShortLivedProfile
	}
	if cfg.Challenge == utils.CertificateChallengeHTTP01 {
		template.DisableTLSALPNChallenge = true
	} else {
		template.DisableHTTPChallenge = true
	}

	issuer := certmagic.NewACMEIssuer(cmCfg, template)
	return issuer, issuer, nil
}

func (m *CertificateManager) Stop() {
	if m == nil {
		return
	}
	m.stateMu.Lock()
	cancel := m.renewalLoopCancel
	m.renewalLoopCancel = nil
	m.stateMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if m.cache != nil {
		m.cache.Stop()
	}
}

func (m *CertificateManager) TLSConfig() *tls.Config {
	if m == nil || !m.enabled || m.certConfig == nil {
		return nil
	}
	tlsConfig := m.certConfig.TLSConfig()
	tlsConfig.NextProtos = normalizeCertificateNextProtos(tlsConfig.NextProtos)
	return tlsConfig
}

func normalizeCertificateNextProtos(items []string) []string {
	out := []string{"http/1.1"}
	seen := map[string]bool{"http/1.1": true}
	for _, item := range items {
		if item == "" || item == "h2" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

func (m *CertificateManager) ObtainNow(ctx context.Context) error {
	if m == nil || !m.enabled || m.certConfig == nil || m.subjectIP == "" {
		return fmt.Errorf("证书管理器未启用")
	}
	return m.obtainManagedCertificate(ctx, "手动触发证书检查", "手动证书检查已完成")
}

func (m *CertificateManager) HTTPValidationHandler(next http.Handler) http.Handler {
	if next == nil {
		next = http.NewServeMux()
	}
	if m == nil || !m.enabled {
		return next
	}
	if m.acmeIssuer != nil {
		return m.acmeIssuer.HTTPChallengeHandler(next)
	}
	if m.zeroIssuer != nil {
		return m.zeroIssuer.HTTPValidationHandler(next)
	}
	return next
}

func (m *CertificateManager) Status(ctx context.Context) CertificateStatus {
	if m == nil {
		return CertificateStatus{}
	}
	m.stateMu.Lock()
	status := CertificateStatus{
		Enabled:              m.enabled,
		RuntimeActive:        m.enabled && m.certConfig != nil,
		SubjectIP:            m.subjectIP,
		Issuer:               string(m.issuer),
		Challenge:            string(m.challenge),
		LastError:            m.lastError,
		LastCheckAt:          m.lastCheckAt,
		LastSuccessAt:        m.lastSuccessAt,
		NextCheckAt:          m.nextCheckAt,
		RetryCount:           m.retryCount,
		Retrying:             m.retrying,
		RenewBeforeDays:      m.renewBeforeDays,
		CheckIntervalMinutes: m.checkIntervalMinutes,
		RetryInitialMinutes:  m.retryInitialMinutes,
		RetryMaxMinutes:      m.retryMaxMinutes,
	}
	m.stateMu.Unlock()
	if !status.RuntimeActive {
		return status
	}
	cert, err := m.certConfig.CacheManagedCertificate(ctx, m.subjectIP)
	if err != nil {
		return status
	}
	if cert.Leaf == nil {
		return status
	}
	status.CertificatePresent = true
	notBefore := cert.Leaf.NotBefore
	notAfter := cert.Leaf.NotAfter
	status.NotBefore = &notBefore
	status.NotAfter = &notAfter
	if time.Now().Before(notAfter) {
		status.RemainingDays = int(time.Until(notAfter).Hours() / 24)
	}
	return status
}

func (m *CertificateManager) Logs(limit int) []CertificateLogEntry {
	if m == nil {
		return nil
	}
	m.logsMu.Lock()
	defer m.logsMu.Unlock()
	if limit <= 0 || limit > certificateLogLimit {
		limit = certificateLogLimit
	}
	if len(m.logs) <= limit {
		out := make([]CertificateLogEntry, len(m.logs))
		copy(out, m.logs)
		return out
	}
	out := make([]CertificateLogEntry, limit)
	copy(out, m.logs[len(m.logs)-limit:])
	return out
}

func (m *CertificateManager) addLog(level, event, message string) {
	entry := CertificateLogEntry{
		Time:      time.Now(),
		Level:     level,
		Event:     event,
		Message:   message,
		SubjectIP: m.subjectIP,
		Issuer:    string(m.issuer),
		Challenge: string(m.challenge),
	}
	log.Printf("[证书] [%s] %s: %s", level, event, message)
	m.logsMu.Lock()
	defer m.logsMu.Unlock()
	m.logs = append(m.logs, entry)
	if len(m.logs) > certificateLogLimit {
		m.logs = m.logs[len(m.logs)-certificateLogLimit:]
	}
}

func (m *CertificateManager) StartRenewalLoop(parent context.Context) {
	if m == nil || !m.enabled || m.certConfig == nil || m.subjectIP == "" {
		return
	}
	m.stateMu.Lock()
	if m.renewalLoopCancel != nil {
		m.stateMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(parent)
	m.renewalLoopCancel = cancel
	runner := m.renewalLoopRunner
	m.stateMu.Unlock()

	if runner == nil {
		runner = m.runSingleRenewalPass
	}
	m.addLog("info", "renewal", "自动续期守护已启动")
	go m.runRenewalLoop(ctx, runner)
}

func (m *CertificateManager) runRenewalLoop(ctx context.Context, runner func(context.Context) time.Duration) {
	for {
		delay := runner(ctx)
		if delay <= 0 {
			delay = time.Duration(m.checkIntervalMinutes) * time.Minute
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			m.addLog("info", "renewal", "自动续期守护已停止")
			return
		case <-timer.C:
		}
	}
}

func (m *CertificateManager) runSingleRenewalPass(ctx context.Context) time.Duration {
	now := time.Now()
	status := m.Status(ctx)
	if shouldRenewCertificate(status.RemainingDays, m.renewBeforeDays, status.CertificatePresent) {
		m.addLog("info", "renewal", "自动续期守护触发证书检查")
		if err := m.obtainManagedCertificate(ctx, "自动续期守护触发证书检查", "自动续期守护证书检查已完成"); err != nil {
			delay := nextCertificateCheckDelay(true, m.currentRetryCount()+1, m.checkIntervalMinutes, m.retryInitialMinutes, m.retryMaxMinutes)
			m.recordRenewalFailure(now, delay, err)
			m.addLog("error", "renewal", fmt.Sprintf("自动续期守护检查失败，将在 %s 后重试: %v", delay, err))
			return delay
		}
		delay := nextCertificateCheckDelay(false, 0, m.checkIntervalMinutes, m.retryInitialMinutes, m.retryMaxMinutes)
		m.recordRenewalSuccess(now, delay)
		return delay
	}

	delay := nextCertificateCheckDelay(false, 0, m.checkIntervalMinutes, m.retryInitialMinutes, m.retryMaxMinutes)
	m.recordRenewalSuccess(now, delay)
	m.addLog("info", "renewal", "证书剩余时间充足，本轮无需续期")
	return delay
}

func (m *CertificateManager) currentRetryCount() int {
	if m == nil {
		return 0
	}
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	return m.retryCount
}

func shouldRenewCertificate(remainingDays int, thresholdDays int, present bool) bool {
	if !present {
		return true
	}
	return remainingDays <= thresholdDays
}

func nextCertificateCheckDelay(retrying bool, retryCount int, normalMinutes int, initialRetryMinutes int, maxRetryMinutes int) time.Duration {
	if !retrying || retryCount <= 0 {
		return time.Duration(normalMinutes) * time.Minute
	}
	delay := initialRetryMinutes
	for i := 1; i < retryCount; i++ {
		delay *= 2
		if delay >= maxRetryMinutes {
			delay = maxRetryMinutes
			break
		}
	}
	return time.Duration(delay) * time.Minute
}

func (m *CertificateManager) recordRenewalFailure(now time.Time, nextDelay time.Duration, err error) {
	if m == nil {
		return
	}
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	m.lastError = err.Error()
	m.retryCount++
	m.retrying = true
	lastCheckAt := now
	nextCheckAt := now.Add(nextDelay)
	m.lastCheckAt = &lastCheckAt
	m.nextCheckAt = &nextCheckAt
}

func (m *CertificateManager) recordRenewalSuccess(now time.Time, nextDelay time.Duration) {
	if m == nil {
		return
	}
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	m.lastError = ""
	m.retryCount = 0
	m.retrying = false
	lastCheckAt := now
	lastSuccessAt := now
	nextCheckAt := now.Add(nextDelay)
	m.lastCheckAt = &lastCheckAt
	m.lastSuccessAt = &lastSuccessAt
	m.nextCheckAt = &nextCheckAt
}

func (m *CertificateManager) obtainManagedCertificate(ctx context.Context, startMessage string, successMessage string) error {
	m.obtainMu.Lock()
	defer m.obtainMu.Unlock()
	m.addLog("info", "obtain", startMessage)
	if err := m.certConfig.ManageSync(ctx, []string{m.subjectIP}); err != nil {
		m.lastError = err.Error()
		m.addLog("error", "obtain", err.Error())
		return err
	}
	m.lastError = ""
	m.addLog("info", "obtain", successMessage)
	return nil
}

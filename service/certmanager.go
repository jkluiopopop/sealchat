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
	Enabled            bool       `json:"enabled"`
	RuntimeActive      bool       `json:"runtimeActive"`
	SubjectIP          string     `json:"subjectIp"`
	Issuer             string     `json:"issuer"`
	Challenge          string     `json:"challenge"`
	CertificatePresent bool       `json:"certificatePresent"`
	NotBefore          *time.Time `json:"notBefore,omitempty"`
	NotAfter           *time.Time `json:"notAfter,omitempty"`
	RemainingDays      int        `json:"remainingDays"`
	LastError          string     `json:"lastError,omitempty"`
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
	if m == nil || m.cache == nil {
		return
	}
	m.cache.Stop()
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
	m.addLog("info", "obtain", "手动触发证书检查")
	if err := m.certConfig.ManageSync(ctx, []string{m.subjectIP}); err != nil {
		m.lastError = err.Error()
		m.addLog("error", "obtain", err.Error())
		return err
	}
	m.lastError = ""
	m.addLog("info", "obtain", "手动证书检查已完成")
	return nil
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
	status := CertificateStatus{
		Enabled:       m.enabled,
		RuntimeActive: m.enabled && m.certConfig != nil,
		SubjectIP:     m.subjectIP,
		Issuer:        string(m.issuer),
		Challenge:     string(m.challenge),
		LastError:     m.lastError,
	}
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

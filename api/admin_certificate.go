package api

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"

	"sealchat/service"
	"sealchat/utils"
)

func AdminCertificateConfigGet(ctx *fiber.Ctx) error {
	cfg := sanitizeConfigForClient(appConfig).Certificate
	return ctx.JSON(fiber.Map{
		"config":          cfg,
		"restartRequired": false,
	})
}

func AdminCertificateConfigUpdate(ctx *fiber.Ctx) error {
	var body struct {
		Config utils.CertificateConfig `json:"config"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		return err
	}
	current := appConfig
	if current == nil {
		current = &utils.AppConfig{}
	}
	incoming := *current
	incoming.Certificate = body.Config
	merged := mergeConfigForWrite(current, &incoming)
	merged.Certificate = utils.NormalizeCertificateConfig(merged.Certificate)
	if err := utils.ValidateCertificateConfig(merged.Certificate); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}

	appConfig = merged
	utils.WriteConfig(appConfig)
	SyncConfigToDB(appConfig, "api")
	return ctx.JSON(fiber.Map{
		"config":          sanitizeConfigForClient(appConfig).Certificate,
		"restartRequired": true,
	})
}

func AdminCertificateStatus(ctx *fiber.Ctx) error {
	status := certificateRuntimeStatus(ctx.Context())
	return ctx.JSON(fiber.Map{
		"status": status,
	})
}

func AdminCertificateLogs(ctx *fiber.Ctx) error {
	limit := ctx.QueryInt("limit", 100)
	if runtimeCertificateManager == nil {
		return ctx.JSON(fiber.Map{
			"items": []service.CertificateLogEntry{},
		})
	}
	return ctx.JSON(fiber.Map{
		"items": runtimeCertificateManager.Logs(limit),
	})
}

func AdminCertificateObtain(ctx *fiber.Ctx) error {
	if runtimeCertificateManager == nil {
		return ctx.Status(fiber.StatusConflict).JSON(fiber.Map{
			"message": "证书配置保存后需重启服务才能签发",
		})
	}
	obtainCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := runtimeCertificateManager.ObtainNow(obtainCtx); err != nil {
		return ctx.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"message": err.Error(),
		})
	}
	return ctx.JSON(fiber.Map{
		"status": runtimeCertificateManager.Status(context.Background()),
	})
}

func certificateRuntimeStatus(ctx context.Context) service.CertificateStatus {
	if runtimeCertificateManager != nil {
		return runtimeCertificateManager.Status(ctx)
	}
	if appConfig == nil {
		return service.CertificateStatus{}
	}
	cfg := utils.NormalizeCertificateConfig(appConfig.Certificate)
	return service.CertificateStatus{
		Enabled:              cfg.Enabled,
		RuntimeActive:        false,
		SubjectIP:            cfg.SubjectIP,
		Issuer:               string(cfg.Issuer),
		Challenge:            string(cfg.Challenge),
		RenewBeforeDays:      cfg.RenewBeforeDays,
		CheckIntervalMinutes: cfg.CheckIntervalMinutes,
		RetryInitialMinutes:  cfg.RetryInitialMinutes,
		RetryMaxMinutes:      cfg.RetryMaxMinutes,
	}
}

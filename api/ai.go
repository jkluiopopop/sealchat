package api

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"sealchat/model"
	aiService "sealchat/service/ai"
	"sealchat/utils"
)

type aiTaskRunner interface {
	Run(ctx context.Context, req aiService.RunRequest) (aiService.RunResult, error)
}

var aiRunnerFactory = func(cfgProvider func() *utils.AppConfig) aiTaskRunner {
	return aiService.NewRunner(cfgProvider, nil)
}

func AICapabilitiesGet(ctx *fiber.Ctx) error {
	user := getCurUser(ctx)
	if user == nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "未登录"})
	}
	if appConfig == nil {
		return ctx.JSON(fiber.Map{"features": []aiService.FeatureCapability{}})
	}
	worldID := strings.TrimSpace(ctx.Query("worldId"))
	features := aiService.AvailableFeatures(appConfig.AI, user.ID, worldID)
	return ctx.JSON(fiber.Map{
		"features": features,
	})
}

func AITaskRun(ctx *fiber.Ctx) error {
	user := getCurUser(ctx)
	if user == nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "未登录"})
	}
	var body struct {
		WorldID   string `json:"worldId"`
		ChannelID string `json:"channelId"`
		Input     string `json:"input"`
		Source    string `json:"source"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		return err
	}
	featureKey := strings.TrimSpace(ctx.Params("featureKey"))
	source := strings.TrimSpace(body.Source)
	now := time.Now()
	var reservation *model.AIQuotaReservationModel
	if appConfig != nil && strings.EqualFold(source, "platform") {
		var err error
		reservation, err = aiService.ReserveQuotaForPlatformRun(appConfig.AI, user.ID, featureKey, body.Input, now)
		if err != nil {
			status := fiber.StatusBadRequest
			switch err.(type) {
			case *aiService.AIQuotaExceededError:
				status = fiber.StatusForbidden
			default:
				if strings.Contains(err.Error(), "pricing") {
					status = fiber.StatusServiceUnavailable
				}
			}
			return ctx.Status(status).JSON(fiber.Map{"message": err.Error()})
		}
	}

	settled := false
	defer func() {
		if !settled && reservation != nil {
			_ = aiService.ReleaseQuotaReservation(reservation.ID)
		}
	}()

	runner := aiRunnerFactory(func() *utils.AppConfig { return appConfig })
	result, err := runner.Run(ctx.Context(), aiService.RunRequest{
		FeatureKey: featureKey,
		UserID:     user.ID,
		WorldID:    strings.TrimSpace(body.WorldID),
		Input:      body.Input,
		Source:     source,
	})
	if err != nil {
		status := fiber.StatusBadRequest
		if strings.Contains(err.Error(), "no ai provider available") {
			status = fiber.StatusServiceUnavailable
		} else if strings.Contains(err.Error(), "unavailable") {
			status = fiber.StatusForbidden
		}
		return ctx.Status(status).JSON(fiber.Map{"message": err.Error()})
	}
	if strings.EqualFold(source, "platform") {
		if !aiService.UsageAvailable(result.Usage) {
			return ctx.Status(fiber.StatusBadGateway).JSON(fiber.Map{"message": "ai usage unavailable"})
		}
		pricing, err := aiService.ResolvePricing(appConfig.AI, result.ProviderID, result.Model)
		if err != nil {
			return ctx.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"message": err.Error()})
		}
		cost := aiService.CalculateUsageCost(result.Usage, *pricing)
		startedAt := result.StartedAt
		finishedAt := result.FinishedAt
		if startedAt.IsZero() {
			startedAt = now
		}
		if finishedAt.IsZero() {
			finishedAt = time.Now()
		}
		usernameSnapshot := strings.TrimSpace(user.Username)
		nicknameSnapshot := strings.TrimSpace(user.Nickname)
		logItem := &model.AIUsageLogModel{
			StringPKBaseModel:    model.StringPKBaseModel{ID: utils.NewID()},
			UserID:               user.ID,
			UsernameSnapshot:     usernameSnapshot,
			NicknameSnapshot:     nicknameSnapshot,
			FeatureKey:           result.FeatureKey,
			ProviderID:           result.ProviderID,
			Model:                result.Model,
			Source:               "platform",
			Status:               "success",
			PromptTokens:         result.Usage.PromptTokens,
			CompletionTokens:     result.Usage.CompletionTokens,
			CacheTokens:          result.Usage.CacheTokens,
			PromptPricePer1M:     cost.PromptPricePer1M,
			CompletionPricePer1M: cost.CompletionPricePer1M,
			CachePricePer1M:      cost.CachePricePer1M,
			PromptCost:           cost.PromptCost,
			CompletionCost:       cost.CompletionCost,
			CacheCost:            cost.CacheCost,
			TotalCost:            cost.TotalCost,
			LatencyMS:            finishedAt.Sub(startedAt).Milliseconds(),
			StartedAt:            startedAt,
			FinishedAt:           finishedAt,
		}
		ledgerItem := &model.AIUsageLedgerModel{
			StringPKBaseModel: model.StringPKBaseModel{ID: utils.NewID()},
			UserID:            user.ID,
			FeatureKey:        result.FeatureKey,
			ProviderID:        result.ProviderID,
			Model:             result.Model,
			BillingDay:        finishedAt.Format("2006-01-02"),
			BillingMonth:      finishedAt.Format("2006-01"),
			PromptTokens:      result.Usage.PromptTokens,
			CompletionTokens:  result.Usage.CompletionTokens,
			CacheTokens:       result.Usage.CacheTokens,
			TotalCost:         cost.TotalCost,
			LogID:             logItem.ID,
		}
		if reservation == nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "ai quota reservation missing"})
		}
		if err := aiService.SettleQuotaReservation(reservation.ID, ledgerItem, logItem); err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
		}
		settled = true
	}
	return ctx.JSON(fiber.Map{
		"featureKey": result.FeatureKey,
		"result":     result.Result,
		"model":      result.Model,
		"providerId": result.ProviderID,
	})
}

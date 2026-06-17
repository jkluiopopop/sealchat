package ai

import (
	"context"
	"errors"
	"strings"
	"time"

	"sealchat/model"
	"sealchat/utils"
)

type TaskRunner interface {
	Run(ctx context.Context, req RunRequest) (RunResult, error)
}

type BilledRunInput struct {
	Config     utils.AIConfig
	User       *model.UserModel
	FeatureKey string
	WorldID    string
	Input      string
	Source     string
	Now        time.Time
	Runner     TaskRunner
}

type BilledRunOutput struct {
	Result RunResult
	Billed bool
}

func RunTaskWithBilling(ctx context.Context, input BilledRunInput) (BilledRunOutput, error) {
	if input.User == nil {
		return BilledRunOutput{}, errors.New("ai user missing")
	}
	if input.Runner == nil {
		return BilledRunOutput{}, errors.New("ai runner unavailable")
	}
	now := input.Now
	if now.IsZero() {
		now = time.Now()
	}
	source := strings.TrimSpace(input.Source)

	var reservation *model.AIQuotaReservationModel
	if strings.EqualFold(source, "platform") {
		var err error
		reservation, err = ReserveQuotaForPlatformRun(input.Config, input.User.ID, input.FeatureKey, input.Input, now)
		if err != nil {
			return BilledRunOutput{}, err
		}
	}

	settled := false
	defer func() {
		if !settled && reservation != nil {
			_ = ReleaseQuotaReservation(reservation.ID)
		}
	}()

	result, err := input.Runner.Run(ctx, RunRequest{
		FeatureKey: strings.TrimSpace(input.FeatureKey),
		UserID:     input.User.ID,
		WorldID:    strings.TrimSpace(input.WorldID),
		Input:      input.Input,
		Source:     source,
	})
	if err != nil {
		return BilledRunOutput{}, err
	}
	if !strings.EqualFold(source, "platform") {
		return BilledRunOutput{Result: result}, nil
	}
	if !UsageAvailable(result.Usage) {
		return BilledRunOutput{}, errors.New("ai usage unavailable")
	}
	pricing, err := ResolvePricing(input.Config, result.ProviderID, result.Model)
	if err != nil {
		return BilledRunOutput{}, err
	}
	cost := CalculateUsageCost(result.Usage, *pricing)
	startedAt := result.StartedAt
	finishedAt := result.FinishedAt
	if startedAt.IsZero() {
		startedAt = now
	}
	if finishedAt.IsZero() {
		finishedAt = time.Now()
	}
	logItem := &model.AIUsageLogModel{
		StringPKBaseModel:    model.StringPKBaseModel{ID: utils.NewID()},
		UserID:               input.User.ID,
		UsernameSnapshot:     strings.TrimSpace(input.User.Username),
		NicknameSnapshot:     strings.TrimSpace(input.User.Nickname),
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
		UserID:            input.User.ID,
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
		return BilledRunOutput{}, errors.New("ai quota reservation missing")
	}
	if err := SettleQuotaReservation(reservation.ID, ledgerItem, logItem); err != nil {
		return BilledRunOutput{}, err
	}
	settled = true
	return BilledRunOutput{Result: result, Billed: true}, nil
}

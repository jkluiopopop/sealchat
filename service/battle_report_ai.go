package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"sealchat/model"
	aiService "sealchat/service/ai"
	"sealchat/utils"
)

type BattleReportSummaryInput struct {
	Title              string
	PeriodStart        time.Time
	PeriodEnd          time.Time
	ContextReportCount int
	Source             string
	AIConfig           utils.AIConfig
	Runner             aiService.TaskRunner
}

type BattleReportSummaryRunOptions struct {
	User     *model.UserModel
	Source   string
	AIConfig utils.AIConfig
	Runner   aiService.TaskRunner
}

func StartBattleReportSummary(ctx context.Context, channelID string, userID string, input BattleReportSummaryInput) (*model.BattleReportModel, error) {
	item, err := CreateBattleReport(channelID, userID, BattleReportInput{
		Title:              input.Title,
		PeriodStart:        input.PeriodStart,
		PeriodEnd:          input.PeriodEnd,
		ContextReportCount: input.ContextReportCount,
		Status:             model.BattleReportStatusGenerating,
		AISource:           input.Source,
		AIFeatureKey:       aiService.FeatureBattleSummary,
	})
	if err != nil {
		return nil, err
	}
	user := model.UserGet(userID)
	if user == nil {
		user = &model.UserModel{StringPKBaseModel: model.StringPKBaseModel{ID: userID}}
	}
	go func() {
		if err := runBattleReportSummaryTask(ctx, item.ID, BattleReportSummaryRunOptions{
			User:     user,
			Source:   input.Source,
			AIConfig: input.AIConfig,
			Runner:   input.Runner,
		}); err != nil {
			_ = markBattleReportSummaryFailed(item.ID, err)
		}
	}()
	return item, nil
}

func runBattleReportSummaryTask(ctx context.Context, reportID string, opts BattleReportSummaryRunOptions) error {
	report, err := loadBattleReport(reportID)
	if err != nil {
		return err
	}
	if opts.User == nil {
		return fmt.Errorf("缺少用户信息")
	}
	messages, err := loadBattleReportMessages(report.ChannelID, report.PeriodStart, report.PeriodEnd)
	if err != nil {
		return err
	}
	if len(messages) == 0 {
		return markBattleReportSummaryFailed(report.ID, fmt.Errorf("所选时间范围内没有可总结的消息"))
	}
	contextReports, err := loadBattleReportContextReports(report)
	if err != nil {
		return err
	}
	prompt := buildBattleReportSummaryPrompt(report, contextReports, messages)
	output, err := aiService.RunTaskWithBilling(ctx, aiService.BilledRunInput{
		Config:     opts.AIConfig,
		User:       opts.User,
		FeatureKey: aiService.FeatureBattleSummary,
		WorldID:    report.WorldID,
		Input:      prompt,
		Source:     opts.Source,
		Runner:     opts.Runner,
	})
	if err != nil {
		return markBattleReportSummaryFailed(report.ID, err)
	}
	result := strings.TrimSpace(output.Result.Result)
	if result == "" {
		return markBattleReportSummaryFailed(report.ID, fmt.Errorf("AI 返回空战报"))
	}
	updates := map[string]interface{}{
		"content":         result,
		"content_preview": model.BuildBattleReportPreview(result, 200),
		"status":          model.BattleReportStatusReady,
		"error_message":   "",
		"ai_source":       strings.TrimSpace(opts.Source),
		"ai_provider_id":  output.Result.ProviderID,
		"ai_model":        output.Result.Model,
		"ai_feature_key":  aiService.FeatureBattleSummary,
	}
	return model.GetDB().Model(&model.BattleReportModel{}).
		Where("id = ? AND is_deleted = ?", report.ID, false).
		Updates(updates).Error
}

func loadBattleReportMessages(channelID string, start time.Time, end time.Time) ([]*model.MessageModel, error) {
	query := model.GetDB().Model(&model.MessageModel{}).
		Where("channel_id = ?", strings.TrimSpace(channelID)).
		Where("is_deleted = ?", false).
		Where("is_revoked = ?", false).
		Preload("Member").
		Preload("User")
	if !start.IsZero() {
		query = query.Where("created_at >= ?", start)
	}
	if !end.IsZero() {
		query = query.Where("created_at <= ?", end)
	}
	query = query.Order("display_order asc").Order("created_at asc")
	var messages []*model.MessageModel
	if err := query.Find(&messages).Error; err != nil {
		return nil, err
	}
	return filterMessagesForBattleReport(messages), nil
}

func filterMessagesForBattleReport(messages []*model.MessageModel) []*model.MessageModel {
	filtered := make([]*model.MessageModel, 0, len(messages))
	for _, msg := range messages {
		if classifyExportMessage(msg, false, false, false).Skip {
			continue
		}
		filtered = append(filtered, msg)
	}
	return filtered
}

func loadBattleReportContextReports(report *model.BattleReportModel) ([]*model.BattleReportModel, error) {
	if report == nil || report.ContextReportCount <= 0 {
		return nil, nil
	}
	var items []*model.BattleReportModel
	err := model.GetDB().
		Where("channel_id = ? AND is_deleted = ? AND status = ? AND id <> ?", report.ChannelID, false, model.BattleReportStatusReady, report.ID).
		Order("sort_order DESC, period_start DESC, created_at DESC").
		Limit(report.ContextReportCount).
		Find(&items).Error
	return items, err
}

func markBattleReportSummaryFailed(reportID string, cause error) error {
	message := "战报总结失败"
	if cause != nil {
		message = cause.Error()
	}
	return model.GetDB().Model(&model.BattleReportModel{}).
		Where("id = ? AND is_deleted = ?", strings.TrimSpace(reportID), false).
		Updates(map[string]interface{}{
			"status":        model.BattleReportStatusFailed,
			"error_message": message,
		}).Error
}

func ResetGeneratingBattleReportsAfterRestart() error {
	return model.GetDB().Model(&model.BattleReportModel{}).
		Where("status = ? AND is_deleted = ?", model.BattleReportStatusGenerating, false).
		Updates(map[string]interface{}{
			"status":        model.BattleReportStatusFailed,
			"error_message": "服务重启，任务未完成，请重试",
		}).Error
}

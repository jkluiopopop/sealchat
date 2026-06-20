package service

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"sealchat/model"
	"sealchat/utils"
)

const battleReportDisplayOrderGap = 1024.0

func GetBattleReportDisplayChannel(sourceChannelID, userID string) (*model.BattleReportDisplayChannelModel, error) {
	sourceChannelID = strings.TrimSpace(sourceChannelID)
	setting, err := loadBattleReportDisplaySettingByAnyChannel(sourceChannelID)
	if err != nil {
		return nil, err
	}
	if setting != nil {
		if err := EnsureBattleReportWorldAccess(userID, setting.WorldID); err != nil {
			return nil, err
		}
		if !setting.Enabled {
			return nil, nil
		}
		return setting, nil
	}
	source, err := loadBattleReportChannel(sourceChannelID)
	if err != nil {
		return nil, err
	}
	if err := EnsureBattleReportWorldAccess(userID, source.WorldID); err != nil {
		return nil, err
	}
	return nil, nil
}

func EnsureBattleReportDisplayChannel(sourceChannelID, userID, name string) (*model.BattleReportDisplayChannelModel, error) {
	sourceChannelID = strings.TrimSpace(sourceChannelID)
	userID = strings.TrimSpace(userID)
	existingSetting, err := loadBattleReportDisplaySettingByAnyChannel(sourceChannelID)
	if err != nil {
		return nil, err
	}
	if existingSetting != nil {
		sourceChannelID = existingSetting.SourceChannelID
	}
	source, err := loadBattleReportChannel(sourceChannelID)
	if err != nil {
		return nil, err
	}
	if err := EnsureBattleReportWorldAccess(userID, source.WorldID); err != nil {
		return nil, err
	}
	displayName := strings.TrimSpace(name)
	if displayName == "" {
		displayName = "战报时间线"
	}

	var setting model.BattleReportDisplayChannelModel
	err = model.GetDB().Where("source_channel_id = ?", sourceChannelID).First(&setting).Error
	if err == nil {
		setting.DisplayName = displayName
		setting.Enabled = true
		setting.Normalize()
		if err := model.GetDB().Save(&setting).Error; err != nil {
			return nil, err
		}
		if strings.TrimSpace(setting.DisplayChannelID) != "" {
			_ = model.GetDB().Model(&model.ChannelModel{}).
				Where("id = ?", setting.DisplayChannelID).
				Updates(map[string]any{"name": setting.DisplayName, "status": model.ChannelStatusActive}).Error
		}
		if err := SyncBattleReportDisplayFromReports(sourceChannelID); err != nil {
			return nil, err
		}
		return &setting, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	displayChannel := ChannelNew(utils.NewID(), "public", displayName, source.WorldID, userID, "")
	if displayChannel == nil || strings.TrimSpace(displayChannel.ID) == "" {
		return nil, fmt.Errorf("战报展示频道创建失败")
	}
	updates := map[string]any{
		"note":                  "战报总结展示频道",
		"default_dice_expr":     source.DefaultDiceExpr,
		"built_in_dice_enabled": source.BuiltInDiceEnabled,
		"bot_feature_enabled":   source.BotFeatureEnabled,
		"status":                model.ChannelStatusActive,
		"sort_order":            source.SortOrder - 1,
	}
	if strings.TrimSpace(source.DefaultDiceExpr) == "" {
		updates["default_dice_expr"] = "d20"
	}
	if err := model.GetDB().Model(&model.ChannelModel{}).
		Where("id = ?", displayChannel.ID).
		Updates(updates).Error; err != nil {
		return nil, err
	}
	if refreshed, err := model.ChannelGet(displayChannel.ID); err == nil && refreshed != nil {
		displayChannel = refreshed
	}

	setting = model.BattleReportDisplayChannelModel{
		StringPKBaseModel: model.StringPKBaseModel{ID: utils.NewID()},
		WorldID:           source.WorldID,
		SourceChannelID:   sourceChannelID,
		DisplayChannelID:  displayChannel.ID,
		DisplayName:       displayName,
		Enabled:           true,
	}
	setting.Normalize()

	if err := model.GetDB().Create(&setting).Error; err != nil {
		return nil, err
	}
	if err := SyncBattleReportDisplayFromReports(sourceChannelID); err != nil {
		return nil, err
	}
	return &setting, nil
}

func DisableBattleReportDisplayChannel(channelID, userID string) error {
	channelID = strings.TrimSpace(channelID)
	userID = strings.TrimSpace(userID)
	setting, err := loadBattleReportDisplaySettingByAnyChannel(channelID)
	if err != nil {
		return err
	}
	if setting == nil {
		return nil
	}
	if err := EnsureBattleReportWorldAccess(userID, setting.WorldID); err != nil {
		return err
	}
	return model.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.BattleReportDisplayChannelModel{}).
			Where("id = ?", setting.ID).
			Update("enabled", false).Error; err != nil {
			return err
		}
		if strings.TrimSpace(setting.DisplayChannelID) != "" {
			if err := tx.Model(&model.ChannelModel{}).
				Where("id = ?", setting.DisplayChannelID).
				Update("status", model.ChannelStatusArchived).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&model.MessageModel{}).
			Where("channel_id = ?", setting.DisplayChannelID).
			Update("is_deleted", true).Error; err != nil {
			return err
		}
		return nil
	})
}

func loadBattleReportDisplaySettingByAnyChannel(channelID string) (*model.BattleReportDisplayChannelModel, error) {
	channelID = strings.TrimSpace(channelID)
	if channelID == "" {
		return nil, nil
	}
	var setting model.BattleReportDisplayChannelModel
	err := model.GetDB().
		Where("source_channel_id = ? OR display_channel_id = ?", channelID, channelID).
		First(&setting).Error
	if err == nil {
		return &setting, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	channel, err := loadBattleReportChannel(channelID)
	if err != nil {
		return nil, err
	}
	err = model.GetDB().
		Where("world_id = ?", channel.WorldID).
		First(&setting).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

func SyncBattleReportDisplayFromReports(sourceChannelID string) error {
	sourceChannelID = strings.TrimSpace(sourceChannelID)
	if sourceChannelID == "" {
		return nil
	}
	setting, err := loadBattleReportDisplaySettingByAnyChannel(sourceChannelID)
	if err != nil {
		return err
	}
	if setting == nil || !setting.Enabled {
		return nil
	}
	sourceChannelID = setting.SourceChannelID
	var reports []model.BattleReportModel
	if err := model.GetDB().
		Where("world_id = ? AND is_deleted = ?", setting.WorldID, false).
		Order("sort_order DESC, period_start DESC, created_at DESC").
		Find(&reports).Error; err != nil {
		return err
	}
	return model.GetDB().Transaction(func(tx *gorm.DB) error {
		seen := map[string]struct{}{}
		total := len(reports)
		for index := range reports {
			report := reports[index]
			displayIndex := total - index
			order := float64(displayIndex) * battleReportDisplayOrderGap
			if err := ensureBattleReportDisplayMessage(tx, setting, &report, order); err != nil {
				return err
			}
			seen[report.ID] = struct{}{}
		}
		var embeds []model.BattleReportDisplayEmbedModel
		if err := tx.Where("display_channel_id = ?", setting.DisplayChannelID).
			Find(&embeds).Error; err != nil {
			return err
		}
		for _, embed := range embeds {
			if _, ok := seen[embed.ReportID]; ok {
				continue
			}
			if err := tx.Model(&model.MessageModel{}).
				Where("id = ? AND channel_id = ?", embed.MessageID, setting.DisplayChannelID).
				Updates(map[string]any{"is_deleted": true}).Error; err != nil {
				return err
			}
			if err := tx.Delete(&model.BattleReportDisplayEmbedModel{}, "id = ?", embed.ID).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func SyncBattleReportOrderFromDisplayMessage(displayChannelID string) error {
	displayChannelID = strings.TrimSpace(displayChannelID)
	if displayChannelID == "" {
		return nil
	}
	var setting model.BattleReportDisplayChannelModel
	if err := model.GetDB().Where("display_channel_id = ? AND enabled = ?", displayChannelID, true).
		First(&setting).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	var rows []struct {
		ReportID     string  `gorm:"column:report_id"`
		MessageOrder float64 `gorm:"column:display_order"`
	}
	if err := model.GetDB().Table("battle_report_display_embeds AS e").
		Select("e.report_id, m.display_order").
		Joins("JOIN messages AS m ON m.id = e.message_id").
		Joins("JOIN battle_reports AS r ON r.id = e.report_id").
		Where("e.display_channel_id = ? AND m.is_deleted = ? AND r.is_deleted = ?", displayChannelID, false, false).
		Order("m.display_order ASC, m.created_at ASC, m.id ASC").
		Scan(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	return model.GetDB().Transaction(func(tx *gorm.DB) error {
		base := len(rows) * 100
		for index := range rows {
			row := rows[len(rows)-1-index]
			sortOrder := base - index*100
			if err := tx.Model(&model.BattleReportModel{}).
				Where("id = ? AND world_id = ? AND is_deleted = ?", row.ReportID, setting.WorldID, false).
				Update("sort_order", sortOrder).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.BattleReportDisplayEmbedModel{}).
				Where("report_id = ? AND display_channel_id = ?", row.ReportID, displayChannelID).
				Update("sort_order", sortOrder).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func ensureBattleReportDisplayMessage(tx *gorm.DB, setting *model.BattleReportDisplayChannelModel, report *model.BattleReportModel, displayOrder float64) error {
	if setting == nil || report == nil {
		return fmt.Errorf("战报展示同步缺少必要数据")
	}
	reportChannelID := strings.TrimSpace(report.ChannelID)
	if reportChannelID == "" {
		reportChannelID = setting.SourceChannelID
	}
	content := buildBattleReportDisplayMessageContent(setting.WorldID, reportChannelID, report.ID)
	var embed model.BattleReportDisplayEmbedModel
	err := tx.Where("report_id = ?", report.ID).First(&embed).Error
	if err == nil {
		embed.SourceChannelID = reportChannelID
		embed.DisplayChannelID = setting.DisplayChannelID
		embed.SortOrder = report.SortOrder
		embed.Normalize()
		if err := tx.Save(&embed).Error; err != nil {
			return err
		}
		return tx.Model(&model.MessageModel{}).
			Where("id = ?", embed.MessageID).
			Updates(map[string]any{
				"channel_id":         setting.DisplayChannelID,
				"content":            content,
				"display_order":      displayOrder,
				"ic_mode":            "ic",
				"is_deleted":         false,
				"sender_member_name": "战报总结",
				"visible_char_count": len([]rune(content)),
			}).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	msg := model.MessageModel{
		StringPKBaseModel: model.StringPKBaseModel{ID: utils.NewID()},
		UserID:            report.CreatorID,
		ChannelID:         setting.DisplayChannelID,
		Content:           content,
		VisibleCharCount:  len([]rune(content)),
		DisplayOrder:      displayOrder,
		ICMode:            "ic",
		SenderMemberName:  "战报总结",
	}
	if strings.TrimSpace(msg.UserID) == "" {
		msg.UserID = report.UpdaterID
	}
	if err := tx.Create(&msg).Error; err != nil {
		return err
	}
	embed = model.BattleReportDisplayEmbedModel{
		StringPKBaseModel: model.StringPKBaseModel{ID: utils.NewID()},
		ReportID:          report.ID,
		SourceChannelID:   reportChannelID,
		DisplayChannelID:  setting.DisplayChannelID,
		MessageID:         msg.ID,
		SortOrder:         report.SortOrder,
	}
	embed.Normalize()
	return tx.Create(&embed).Error
}

func buildBattleReportDisplayMessageContent(worldID, sourceChannelID, reportID string) string {
	return fmt.Sprintf("/#/%s/%s?battleReport=%s", worldID, sourceChannelID, reportID)
}

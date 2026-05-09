package service

import (
	"fmt"
	"errors"
	"strings"
	"sync"

	"sealchat/model"
	"sealchat/pm"
	"sealchat/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AudioQuotaSource string

const (
	AudioQuotaSourceDefault        AudioQuotaSource = "default"
	AudioQuotaSourceOverride       AudioQuotaSource = "override"
	AudioQuotaSourceAdminUnlimited AudioQuotaSource = "admin-unlimited"
)

type AudioQuotaSummary struct {
	Limited        bool             `json:"limited"`
	QuotaBytes     *int64           `json:"quotaBytes"`
	UsedBytes      int64            `json:"usedBytes"`
	RemainingBytes *int64           `json:"remainingBytes"`
	UsagePercent   *float64         `json:"usagePercent"`
	Source         AudioQuotaSource `json:"source"`
}

type AudioQuotaExceededError struct {
	UsedBytes     int64
	QuotaBytes    int64
	IncomingBytes int64
}

type AdminAudioQuotaItem struct {
	UserID         string            `json:"userId"`
	Username       string            `json:"username"`
	Nickname       string            `json:"nickname"`
	HasOverride    bool              `json:"hasOverride"`
	QuotaMB        int64             `json:"quotaMB"`
	UsedBytes      int64             `json:"usedBytes"`
	Limited        bool              `json:"limited"`
	QuotaBytes     *int64            `json:"quotaBytes"`
	RemainingBytes *int64            `json:"remainingBytes"`
	UsagePercent   *float64          `json:"usagePercent"`
	Source         AudioQuotaSource  `json:"source"`
	UpdatedBy      string            `json:"updatedBy,omitempty"`
}

type AdminAudioQuotaListResult struct {
	Items    []AdminAudioQuotaItem `json:"items"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"pageSize"`
	Total    int64                 `json:"total"`
}

func (e *AudioQuotaExceededError) Error() string {
	if e == nil {
		return "音频容量不足"
	}
	return fmt.Sprintf(
		"音频容量不足：当前已用 %s / %s，本次上传 %s",
		formatAudioQuotaBytes(e.UsedBytes),
		formatAudioQuotaBytes(e.QuotaBytes),
		formatAudioQuotaBytes(e.IncomingBytes),
	)
}

var audioQuotaUserLocks sync.Map

func withAudioQuotaUserLock(userID string, fn func() error) error {
	normalized := strings.TrimSpace(userID)
	if normalized == "" {
		return fn()
	}
	lockValue, _ := audioQuotaUserLocks.LoadOrStore(normalized, &sync.Mutex{})
	mu := lockValue.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()
	return fn()
}

func GetAudioUsedBytes(userID string) (int64, error) {
	normalized := strings.TrimSpace(userID)
	if normalized == "" {
		return 0, nil
	}
	var total int64
	err := model.GetDB().
		Model(&model.AudioAsset{}).
		Where("created_by = ? AND deleted_at IS NULL", normalized).
		Select("COALESCE(SUM(size), 0)").
		Scan(&total).Error
	return total, err
}

func GetAudioQuotaSummary(userID string) (*AudioQuotaSummary, error) {
	normalized := strings.TrimSpace(userID)
	if normalized == "" {
		return &AudioQuotaSummary{
			Limited:   false,
			UsedBytes: 0,
			Source:    AudioQuotaSourceAdminUnlimited,
		}, nil
	}
	usedBytes, err := GetAudioUsedBytes(normalized)
	if err != nil {
		return nil, err
	}
	return buildAudioQuotaSummary(normalized, usedBytes)
}

func EnsureAudioQuotaForIncoming(userID string, incomingBytes int64) (*AudioQuotaSummary, error) {
	normalized := strings.TrimSpace(userID)
	if normalized == "" {
		return &AudioQuotaSummary{
			Limited:   false,
			UsedBytes: 0,
			Source:    AudioQuotaSourceAdminUnlimited,
		}, nil
	}
	var summary *AudioQuotaSummary
	err := withAudioQuotaUserLock(normalized, func() error {
		usedBytes, err := GetAudioUsedBytes(normalized)
		if err != nil {
			return err
		}
		summary, err = buildAudioQuotaSummary(normalized, usedBytes)
		if err != nil {
			return err
		}
		if summary.Limited && summary.QuotaBytes != nil && usedBytes+incomingBytes > *summary.QuotaBytes {
			return &AudioQuotaExceededError{
				UsedBytes:     usedBytes,
				QuotaBytes:    *summary.QuotaBytes,
				IncomingBytes: incomingBytes,
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return summary, nil
}

func buildAudioQuotaSummary(userID string, usedBytes int64) (*AudioQuotaSummary, error) {
	if override, err := getAudioUserQuotaOverride(userID); err != nil {
		return nil, err
	} else if override != nil {
		return buildLimitedAudioQuotaSummary(usedBytes, override.QuotaMB, AudioQuotaSourceOverride), nil
	}
	if pm.CanWithSystemRole(userID, pm.PermModAdmin) {
		return &AudioQuotaSummary{
			Limited:   false,
			UsedBytes: usedBytes,
			Source:    AudioQuotaSourceAdminUnlimited,
		}, nil
	}
	quotaMB := int64(150)
	if cfg := utils.GetConfig(); cfg != nil && cfg.Audio.UserQuotaMB > 0 {
		quotaMB = cfg.Audio.UserQuotaMB
	}
	return buildLimitedAudioQuotaSummary(usedBytes, quotaMB, AudioQuotaSourceDefault), nil
}

func buildLimitedAudioQuotaSummary(usedBytes, quotaMB int64, source AudioQuotaSource) *AudioQuotaSummary {
	quotaBytes := quotaMB * 1024 * 1024
	remainingBytes := quotaBytes - usedBytes
	if remainingBytes < 0 {
		remainingBytes = 0
	}
	usagePercent := 0.0
	if quotaBytes > 0 {
		usagePercent = float64(usedBytes) / float64(quotaBytes) * 100
		if usagePercent < 0 {
			usagePercent = 0
		}
	}
	return &AudioQuotaSummary{
		Limited:        true,
		QuotaBytes:     &quotaBytes,
		UsedBytes:      usedBytes,
		RemainingBytes: &remainingBytes,
		UsagePercent:   &usagePercent,
		Source:         source,
	}
}

func getAudioUserQuotaOverride(userID string) (*model.AudioUserQuotaOverride, error) {
	var override model.AudioUserQuotaOverride
	err := model.GetDB().
		Where("user_id = ?", userID).
		Limit(1).
		Find(&override).Error
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(override.ID) == "" {
		return nil, nil
	}
	return &override, nil
}

func formatAudioQuotaBytes(value int64) string {
	if value < 1024 {
		return fmt.Sprintf("%d B", value)
	}
	mb := float64(value) / 1024 / 1024
	if mb < 1024 {
		return fmt.Sprintf("%.1f MB", mb)
	}
	gb := mb / 1024
	return fmt.Sprintf("%.2f GB", gb)
}

func UpsertAudioUserQuotaOverride(userID, updatedBy string, quotaMB int64) (*model.AudioUserQuotaOverride, error) {
	userID = strings.TrimSpace(userID)
	updatedBy = strings.TrimSpace(updatedBy)
	if userID == "" {
		return nil, errors.New("用户ID不能为空")
	}
	if quotaMB <= 0 {
		return nil, errors.New("配额必须大于 0")
	}
	if exists, err := userExists(userID); err != nil {
		return nil, err
	} else if !exists {
		return nil, gorm.ErrRecordNotFound
	}
	record := &model.AudioUserQuotaOverride{
		StringPKBaseModel: model.StringPKBaseModel{ID: utils.NewID()},
		UserID:            userID,
		QuotaMB:           quotaMB,
		UpdatedBy:         updatedBy,
	}
	if err := model.GetDB().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"quota_mb", "updated_by", "updated_at"}),
	}).Create(record).Error; err != nil {
		return nil, err
	}
	return getAudioUserQuotaOverride(userID)
}

func DeleteAudioUserQuotaOverride(userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("用户ID不能为空")
	}
	return model.GetDB().Where("user_id = ?", userID).Delete(&model.AudioUserQuotaOverride{}).Error
}

func GetAdminAudioQuotaDetail(userID string) (*AdminAudioQuotaItem, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errors.New("用户ID不能为空")
	}
	var user model.UserModel
	if err := model.GetDB().Where("id = ?", userID).Limit(1).Find(&user).Error; err != nil {
		return nil, err
	}
	if strings.TrimSpace(user.ID) == "" {
		return nil, gorm.ErrRecordNotFound
	}
	override, err := getAudioUserQuotaOverride(userID)
	if err != nil {
		return nil, err
	}
	summary, err := GetAudioQuotaSummary(userID)
	if err != nil {
		return nil, err
	}
	item := &AdminAudioQuotaItem{
		UserID:         user.ID,
		Username:       user.Username,
		Nickname:       user.Nickname,
		UsedBytes:      summary.UsedBytes,
		Limited:        summary.Limited,
		QuotaBytes:     summary.QuotaBytes,
		RemainingBytes: summary.RemainingBytes,
		UsagePercent:   summary.UsagePercent,
		Source:         summary.Source,
	}
	if override != nil {
		item.HasOverride = true
		item.QuotaMB = override.QuotaMB
		item.UpdatedBy = override.UpdatedBy
	}
	return item, nil
}

func ListAdminAudioQuotaOverrides(page, pageSize int, query string) (*AdminAudioQuotaListResult, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}
	query = strings.TrimSpace(query)
	db := model.GetDB().Model(&model.AudioUserQuotaOverride{})
	if query != "" {
		like := "%" + query + "%"
		userQuery := model.GetDB().Model(&model.UserModel{}).Select("id").Where("id LIKE ? OR username LIKE ? OR nickname LIKE ?", like, like, like)
		db = db.Where("user_id IN (?)", userQuery)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []model.AudioUserQuotaOverride
	if err := db.Order("updated_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]AdminAudioQuotaItem, 0, len(rows))
	for _, row := range rows {
		item, err := GetAdminAudioQuotaDetail(row.UserID)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return &AdminAudioQuotaListResult{
		Items:    items,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}, nil
}

func userExists(userID string) (bool, error) {
	var count int64
	if err := model.GetDB().Model(&model.UserModel{}).Where("id = ?", userID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

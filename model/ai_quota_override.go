package model

import "strings"

type AIUserQuotaOverrideModel struct {
	StringPKBaseModel
	UserID        string   `json:"userId" gorm:"size:100;uniqueIndex"`
	DailyLimit    *float64 `json:"dailyLimit,omitempty"`
	MonthlyLimit  *float64 `json:"monthlyLimit,omitempty"`
	LifetimeLimit *float64 `json:"lifetimeLimit,omitempty"`
	UpdatedBy     string   `json:"updatedBy" gorm:"size:100"`
}

func (*AIUserQuotaOverrideModel) TableName() string {
	return "ai_user_quota_overrides"
}

func AIUserQuotaOverrideGet(userID string) (*AIUserQuotaOverrideModel, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, nil
	}
	var item AIUserQuotaOverrideModel
	if err := db.Where("user_id = ?", userID).Limit(1).Find(&item).Error; err != nil {
		return nil, err
	}
	if strings.TrimSpace(item.ID) == "" {
		return nil, nil
	}
	return &item, nil
}

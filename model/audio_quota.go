package model

type AudioUserQuotaOverride struct {
	StringPKBaseModel
	UserID    string `json:"userId" gorm:"uniqueIndex"`
	QuotaMB   int64  `json:"quotaMB"`
	UpdatedBy string `json:"updatedBy"`
}

func (*AudioUserQuotaOverride) TableName() string {
	return "audio_user_quota_overrides"
}

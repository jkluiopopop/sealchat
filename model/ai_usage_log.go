package model

import "time"

type AIUsageLogModel struct {
	StringPKBaseModel
	UserID               string    `json:"userId" gorm:"size:100;index"`
	UsernameSnapshot     string    `json:"usernameSnapshot" gorm:"size:128;index"`
	NicknameSnapshot     string    `json:"nicknameSnapshot" gorm:"size:128"`
	FeatureKey           string    `json:"featureKey" gorm:"size:64;index"`
	ProviderID           string    `json:"providerId" gorm:"size:64;index"`
	Model                string    `json:"model" gorm:"size:128;index"`
	Source               string    `json:"source" gorm:"size:32;index"`
	Status               string    `json:"status" gorm:"size:32;index"`
	PromptTokens         int64     `json:"promptTokens"`
	CompletionTokens     int64     `json:"completionTokens"`
	CacheTokens          int64     `json:"cacheTokens"`
	PromptPricePer1M     float64   `json:"promptPricePer1M"`
	CompletionPricePer1M float64   `json:"completionPricePer1M"`
	CachePricePer1M      float64   `json:"cachePricePer1M"`
	PromptCost           float64   `json:"promptCost"`
	CompletionCost       float64   `json:"completionCost"`
	CacheCost            float64   `json:"cacheCost"`
	TotalCost            float64   `json:"totalCost" gorm:"index"`
	LatencyMS            int64     `json:"latencyMs"`
	StartedAt            time.Time `json:"startedAt" gorm:"index"`
	FinishedAt           time.Time `json:"finishedAt" gorm:"index"`
	ErrorMessage         string    `json:"errorMessage" gorm:"size:512"`
}

func (*AIUsageLogModel) TableName() string {
	return "ai_usage_logs"
}

func AIUsageLogCleanupBefore(cutoff time.Time) (int64, error) {
	if cutoff.IsZero() {
		return 0, nil
	}
	tx := db.Where("finished_at < ?", cutoff).Delete(&AIUsageLogModel{})
	return tx.RowsAffected, tx.Error
}

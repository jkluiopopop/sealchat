package model

import "time"

type AIQuotaReservationModel struct {
	StringPKBaseModel
	UserID       string    `json:"userId" gorm:"size:100;index"`
	FeatureKey   string    `json:"featureKey" gorm:"size:64;index"`
	ProviderID   string    `json:"providerId" gorm:"size:64;index"`
	Model        string    `json:"model" gorm:"size:128;index"`
	ReservedCost float64   `json:"reservedCost"`
	Status       string    `json:"status" gorm:"size:32;index"`
	ExpiresAt    time.Time `json:"expiresAt" gorm:"index"`
}

func (*AIQuotaReservationModel) TableName() string {
	return "ai_quota_reservations"
}

func AIQuotaReservationCleanupExpired(now time.Time) (int64, error) {
	if now.IsZero() {
		now = time.Now()
	}
	tx := db.Where("status = ? AND expires_at < ?", "active", now).
		Delete(&AIQuotaReservationModel{})
	return tx.RowsAffected, tx.Error
}

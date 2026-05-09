package model

import (
	"strings"
	"time"
)

type WebhookEventLogModel struct {
	Seq           int64     `json:"seq" gorm:"primaryKey;autoIncrement"`
	ChannelID     string    `json:"channelId" gorm:"size:100;index"`
	Type          string    `json:"type" gorm:"size:32;index"`
	MessageID     string    `json:"messageId" gorm:"size:100;index"`
	IntegrationID string    `json:"integrationId" gorm:"size:100;index"`
	Source        string    `json:"source" gorm:"size:32;index"`
	ExternalID    string    `json:"externalId" gorm:"size:128"`
	PayloadJSON   string    `json:"payloadJson" gorm:"type:text"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (*WebhookEventLogModel) TableName() string {
	return "webhook_event_logs"
}

func WebhookEventLogAppend(channelID, eventType, messageID, integrationID, source, externalID string, payloadJSON string) error {
	channelID = strings.TrimSpace(channelID)
	eventType = strings.TrimSpace(eventType)
	messageID = strings.TrimSpace(messageID)
	integrationID = strings.TrimSpace(integrationID)
	source = strings.TrimSpace(source)
	externalID = strings.TrimSpace(externalID)
	if channelID == "" || eventType == "" {
		return nil
	}
	item := &WebhookEventLogModel{
		ChannelID:     channelID,
		Type:          eventType,
		MessageID:     messageID,
		IntegrationID: integrationID,
		Source:        source,
		ExternalID:    externalID,
		PayloadJSON:   payloadJSON,
		CreatedAt:     time.Now(),
	}
	return db.Create(item).Error
}

func WebhookEventLogAppendForMessage(channelID, eventType, messageID string) error {
	channelID = strings.TrimSpace(channelID)
	eventType = strings.TrimSpace(eventType)
	messageID = strings.TrimSpace(messageID)
	if channelID == "" || eventType == "" || messageID == "" {
		return nil
	}
	ref, err := MessageExternalRefGetFirstByMessageID(messageID)
	if err != nil {
		return err
	}
	integrationID := ""
	source := ""
	externalID := ""
	if ref != nil {
		integrationID = strings.TrimSpace(ref.IntegrationID)
		source = strings.TrimSpace(ref.Source)
		externalID = strings.TrimSpace(ref.ExternalID)
	}
	return WebhookEventLogAppend(channelID, eventType, messageID, integrationID, source, externalID, "")
}

func WebhookEventLogCleanupBefore(cutoff time.Time) (int64, error) {
	if cutoff.IsZero() {
		return 0, nil
	}
	tx := db.Where("created_at < ?", cutoff).Delete(&WebhookEventLogModel{})
	return tx.RowsAffected, tx.Error
}

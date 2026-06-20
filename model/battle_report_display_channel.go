package model

import "strings"

type BattleReportDisplayChannelModel struct {
	StringPKBaseModel
	WorldID          string `json:"worldId" gorm:"size:100;index"`
	SourceChannelID  string `json:"sourceChannelId" gorm:"size:100;uniqueIndex"`
	DisplayChannelID string `json:"displayChannelId" gorm:"size:100;uniqueIndex"`
	DisplayName      string `json:"displayName" gorm:"size:255"`
	Enabled          bool   `json:"enabled" gorm:"default:true;index"`
}

func (*BattleReportDisplayChannelModel) TableName() string {
	return "battle_report_display_channels"
}

func (m *BattleReportDisplayChannelModel) Normalize() {
	if m == nil {
		return
	}
	m.WorldID = strings.TrimSpace(m.WorldID)
	m.SourceChannelID = strings.TrimSpace(m.SourceChannelID)
	m.DisplayChannelID = strings.TrimSpace(m.DisplayChannelID)
	m.DisplayName = strings.TrimSpace(m.DisplayName)
	if m.DisplayName == "" {
		m.DisplayName = "战报时间线"
	}
}

type BattleReportDisplayEmbedModel struct {
	StringPKBaseModel
	ReportID         string `json:"reportId" gorm:"size:100;uniqueIndex;index"`
	SourceChannelID  string `json:"sourceChannelId" gorm:"size:100;index"`
	DisplayChannelID string `json:"displayChannelId" gorm:"size:100;index:idx_battle_report_display_embed_channel_order,priority:1"`
	MessageID        string `json:"messageId" gorm:"size:100;uniqueIndex;index"`
	SortOrder        int    `json:"sortOrder" gorm:"default:0;index:idx_battle_report_display_embed_channel_order,priority:2"`
}

func (*BattleReportDisplayEmbedModel) TableName() string {
	return "battle_report_display_embeds"
}

func (m *BattleReportDisplayEmbedModel) Normalize() {
	if m == nil {
		return
	}
	m.ReportID = strings.TrimSpace(m.ReportID)
	m.SourceChannelID = strings.TrimSpace(m.SourceChannelID)
	m.DisplayChannelID = strings.TrimSpace(m.DisplayChannelID)
	m.MessageID = strings.TrimSpace(m.MessageID)
}

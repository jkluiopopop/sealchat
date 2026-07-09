package model

import (
	"strings"
)

type CharacterCardTemplateModel struct {
	StringPKBaseModel
	UserID               string `json:"userId" gorm:"size:100;index:idx_character_card_template_user_sheet,priority:1"`
	Name                 string `json:"name" gorm:"size:100"`
	SheetType            string `json:"sheetType" gorm:"size:32;index:idx_character_card_template_user_sheet,priority:2"`
	Content              string `json:"content" gorm:"type:text"`
	DefaultBadgeTemplate string `json:"defaultBadgeTemplate" gorm:"size:512"`
	IsGlobalDefault      bool   `json:"isGlobalDefault" gorm:"index"`
	IsSheetDefault       bool   `json:"isSheetDefault" gorm:"index:idx_character_card_template_sheet_default,priority:1"`
}

func (*CharacterCardTemplateModel) TableName() string {
	return "character_card_templates"
}

func CharacterCardTemplateList(userID string, sheetType string) ([]*CharacterCardTemplateModel, error) {
	var items []*CharacterCardTemplateModel
	q := db.Where("user_id = ?", userID)
	if trimmed := strings.TrimSpace(sheetType); trimmed != "" {
		q = q.Where("sheet_type = ?", trimmed)
	}
	err := q.Order("is_global_default desc").
		Order("is_sheet_default desc").
		Order("updated_at desc").
		Find(&items).Error
	return items, err
}

func CharacterCardTemplateGetByID(id string) (*CharacterCardTemplateModel, error) {
	item := &CharacterCardTemplateModel{}
	if err := db.Where("id = ?", id).Take(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func CharacterCardTemplateCreate(item *CharacterCardTemplateModel) error {
	return db.Create(item).Error
}

func CharacterCardTemplateUpdate(id string, values map[string]any) error {
	if len(values) == 0 {
		return nil
	}
	return db.Model(&CharacterCardTemplateModel{}).Where("id = ?", id).Updates(values).Error
}

func CharacterCardTemplateDelete(id string) error {
	return db.Where("id = ?", id).Delete(&CharacterCardTemplateModel{}).Error
}

func CharacterCardTemplateClearGlobalDefault(userID string) error {
	return db.Model(&CharacterCardTemplateModel{}).
		Where("user_id = ?", userID).
		Where("is_global_default = ?", true).
		Update("is_global_default", false).Error
}

func CharacterCardTemplateClearSheetDefault(userID string, sheetType string) error {
	return db.Model(&CharacterCardTemplateModel{}).
		Where("user_id = ?", userID).
		Where("sheet_type = ?", strings.TrimSpace(sheetType)).
		Where("is_sheet_default = ?", true).
		Update("is_sheet_default", false).Error
}

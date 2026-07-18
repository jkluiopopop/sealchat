package service

import (
	"encoding/json"
	"strings"

	"sealchat/model"
	"sealchat/pm"
	"sealchat/protocol"
)

func WorldTheaterPresentationTemplateSet(worldID, actorID string, template protocol.WorldTheaterPresentationTemplate) (*model.WorldModel, error) {
	worldID = strings.TrimSpace(worldID)
	actorID = strings.TrimSpace(actorID)
	if worldID == "" || actorID == "" {
		return nil, ErrWorldPermission
	}
	if !pm.CanWithSystemRole(actorID, pm.PermModAdmin) && !IsWorldAdmin(worldID, actorID) {
		return nil, ErrWorldPermission
	}
	if err := protocol.ValidateWorldTheaterPresentationTemplate(template); err != nil {
		return nil, err
	}
	if template.Dialogue != nil && template.Dialogue.Frame != nil {
		var asset model.TheaterAppearanceAssetModel
		if err := model.GetDB().Where("id = ? AND deleted_at IS NULL", template.Dialogue.Frame.Media.AssetID).Limit(1).Find(&asset).Error; err != nil {
			return nil, err
		}
		if asset.ID == "" || asset.Status != "ready" || asset.Purpose != "dialogue-frame" || !theaterMediaRefMatchesAsset(template.Dialogue.Frame.Media, asset) {
			return nil, newTheaterError(TheaterAppearanceAssetErrorInvalid, "世界默认对话框资源无效", 400, nil)
		}
		var channel model.ChannelModel
		if err := model.GetDB().Where("id = ? AND world_id = ?", asset.ChannelID, worldID).Limit(1).Find(&channel).Error; err != nil {
			return nil, err
		}
		if channel.ID == "" {
			return nil, newTheaterError(TheaterAppearanceAssetErrorScopeMismatch, "对话框资源不属于当前世界", 400, nil)
		}
	}
	raw, err := json.Marshal(template)
	if err != nil {
		return nil, err
	}
	result := model.GetDB().Model(&model.WorldModel{}).
		Where("id = ? AND status = ?", worldID, "active").
		Update("theater_presentation_template_json", string(raw))
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrWorldNotFound
	}
	return GetWorldByID(worldID)
}

func WorldTheaterPresentationDefaultsForChannel(channelID string) *protocol.TheaterPresentation {
	channel, err := model.ChannelGet(strings.TrimSpace(channelID))
	if err != nil || channel == nil || strings.TrimSpace(channel.WorldID) == "" {
		return nil
	}
	world, err := GetWorldByID(channel.WorldID)
	if err != nil || world == nil {
		return nil
	}
	template := world.GetTheaterPresentationTemplate()
	if template.Portrait == nil && template.Speaker == nil && template.Content == nil && template.Dialogue == nil {
		return nil
	}
	defaults := protocol.ApplyWorldTheaterPresentationTemplate(protocol.DefaultTheaterPresentation(), template)
	return &defaults
}

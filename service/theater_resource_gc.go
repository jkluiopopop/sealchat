package service

import (
	"context"
	"time"

	"gorm.io/gorm"

	"sealchat/model"
)

func RecalculateTheaterResourceReferences(roomID string) error {
	return model.GetDB().Transaction(func(tx *gorm.DB) error {
		return recalculateTheaterResourceReferences(tx, roomID)
	})
}

func RunTheaterResourceGC(ctx context.Context, grace time.Duration, limit int) (int, error) {
	if grace <= 0 {
		grace = 7 * 24 * time.Hour
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	cutoff := time.Now().Add(-grace)
	var resources []model.TheaterResourceModel
	if err := model.GetDB().Where("status = ? AND reference_count = 0 AND deleted_at IS NOT NULL AND deleted_at < ?", "deleting", cutoff).Limit(limit).Find(&resources).Error; err != nil {
		return 0, err
	}
	deleted := 0
	for _, resource := range resources {
		select {
		case <-ctx.Done():
			return deleted, ctx.Err()
		default:
		}
		if err := deleteTheaterResourcePhysical(ctx, resource); err != nil {
			return deleted, err
		}
		deleted++
	}
	return deleted, nil
}

func deleteTheaterResourcePhysical(ctx context.Context, resource model.TheaterResourceModel) error {
	var variants []model.TheaterResourceVariantModel
	if err := model.GetDB().Where("resource_id = ?", resource.ID).Find(&variants).Error; err != nil {
		return err
	}
	attachmentIDs := []string{resource.AttachmentID}
	for _, variant := range variants {
		attachmentIDs = append(attachmentIDs, variant.AttachmentID)
	}
	return model.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("resource_id = ?", resource.ID).Delete(&model.TheaterResourceVariantModel{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("resource_id = ?", resource.ID).Delete(&model.TheaterResourceJobModel{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("id = ? AND reference_count = 0", resource.ID).Delete(&model.TheaterResourceModel{}).Error; err != nil {
			return err
		}
		for _, attachmentID := range attachmentIDs {
			var otherRefs int64
			if err := tx.Model(&model.TheaterResourceModel{}).Where("attachment_id = ?", attachmentID).Count(&otherRefs).Error; err != nil {
				return err
			}
			var variantRefs int64
			if err := tx.Model(&model.TheaterResourceVariantModel{}).Where("attachment_id = ?", attachmentID).Count(&variantRefs).Error; err != nil {
				return err
			}
			if otherRefs+variantRefs == 0 {
				var attachment model.AttachmentModel
				if err := tx.Where("id = ?", attachmentID).Limit(1).Find(&attachment).Error; err != nil {
					return err
				}
				if attachment.ID != "" {
					if manager := GetStorageManager(); manager != nil {
						_ = manager.Delete(ctx, convertModelToBackend(attachment.StorageType), attachment.ObjectKey)
					}
					if err := tx.Unscoped().Delete(&attachment).Error; err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
}

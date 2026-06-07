package service

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"sealchat/model"
)

var (
	ErrAdminUserDeleteNotFound      = errors.New("user not found")
	ErrAdminUserDeleteAlreadyGone   = errors.New("user already deleted")
	ErrAdminUserDeleteNotDisabled   = errors.New("user not disabled")
	ErrAdminUserDeleteSelf          = errors.New("cannot delete self")
	ErrAdminUserDeleteBot           = errors.New("cannot delete bot user")
	ErrAdminUserDeleteLastAdmin     = errors.New("cannot delete last system admin")
	ErrAdminUserDeleteOwnsWorld     = errors.New("user still owns active worlds")
	ErrAdminUserDeleteOperatorEmpty = errors.New("operator user id is empty")
)

func AdminUserSoftDelete(userID, operatorID string) error {
	userID = strings.TrimSpace(userID)
	operatorID = strings.TrimSpace(operatorID)
	if userID == "" {
		return ErrAdminUserDeleteNotFound
	}
	if operatorID == "" {
		return ErrAdminUserDeleteOperatorEmpty
	}
	if userID == operatorID {
		return ErrAdminUserDeleteSelf
	}

	db := model.GetDB()
	var target model.UserModel
	if err := db.Where("id = ?", userID).Limit(1).Find(&target).Error; err != nil {
		return err
	}
	if strings.TrimSpace(target.ID) == "" {
		return ErrAdminUserDeleteNotFound
	}
	if target.DeletedAt != nil {
		return ErrAdminUserDeleteAlreadyGone
	}
	if target.IsBot {
		return ErrAdminUserDeleteBot
	}
	if !target.Disabled {
		return ErrAdminUserDeleteNotDisabled
	}

	roleIDs, err := model.UserRoleMappingListByUserID(userID, "", "system")
	if err != nil {
		return err
	}
	for _, roleID := range roleIDs {
		if roleID != "sys-admin" {
			continue
		}
		adminIDs, err := model.UserRoleMappingUserIdListByRoleId("sys-admin")
		if err != nil {
			return err
		}
		activeAdmins := 0
		for _, adminID := range adminIDs {
			var admin model.UserModel
			if err := db.Where("id = ?", adminID).Limit(1).Find(&admin).Error; err != nil {
				return err
			}
			if strings.TrimSpace(admin.ID) == "" || admin.DeletedAt != nil {
				continue
			}
			activeAdmins++
		}
		if activeAdmins <= 1 {
			return ErrAdminUserDeleteLastAdmin
		}
		break
	}

	var ownedWorldCount int64
	if err := db.Model(&model.WorldModel{}).
		Where("owner_id = ? AND status = ?", userID, "active").
		Count(&ownedWorldCount).Error; err != nil {
		return err
	}
	if ownedWorldCount > 0 {
		return ErrAdminUserDeleteOwnsWorld
	}

	deletedAssets := make([]*model.AudioAsset, 0)
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&model.AccessTokenModel{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&model.WorldMemberModel{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&model.MemberModel{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&model.UserRoleMappingModel{}).Error; err != nil {
			return err
		}

		var assetIDs []string
		if err := tx.Model(&model.AudioAsset{}).
			Where("created_by = ? AND deleted_at IS NULL", userID).
			Pluck("id", &assetIDs).Error; err != nil {
			return err
		}
		for _, assetID := range assetIDs {
			asset, _, _, err := audioDetachAndDeleteAssetTx(tx, assetID, true)
			if err != nil {
				return err
			}
			if asset != nil {
				deletedAssets = append(deletedAssets, asset)
			}
		}

		now := time.Now()
		return tx.Model(&model.UserModel{}).
			Where("id = ? AND deleted_at IS NULL", userID).
			Updates(map[string]any{
				"disabled":   true,
				"deleted_at": now,
				"updated_at": now,
			}).Error
	})
	if err != nil {
		return err
	}

	for _, asset := range deletedAssets {
		audioDeleteAssetObjects(asset)
	}
	return nil
}

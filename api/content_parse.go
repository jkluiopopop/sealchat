package api

import (
	"sealchat/model"
	"sealchat/utils"
)

func (ctx *ChatContext) TagCheck(ChannelID, msgId, text string) {
	db := model.GetDB()
	for receiverID := range collectMentionTargetIDsFromContent(text) {
		if receiverID == "" {
			continue
		}
		mention := model.MentionModel{
			StringPKBaseModel: model.StringPKBaseModel{
				ID: utils.NewID(),
			},
			ReceiverId:  receiverID,
			SenderId:    ctx.User.ID,
			LocPostType: "channel",
			LocPostID:   ChannelID,
			RelatedType: "message",
			RelatedID:   msgId,
		}
		db.Create(&mention)
	}
}

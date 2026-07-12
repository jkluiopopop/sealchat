package api

import (
	"errors"
	"strings"
	"time"

	"sealchat/service"
)

type quickLoginApprovePayload struct {
	RequestID string `json:"request_id"`
}

type quickLoginDenyPayload struct {
	RequestID string `json:"request_id"`
}

func apiQuickLoginApprove(ctx *ChatContext, data *quickLoginApprovePayload) (any, error) {
	if ctx == nil || ctx.User == nil || ctx.ConnInfo == nil || ctx.Conn == nil {
		return nil, errors.New("未登录")
	}
	requestID := ""
	if data != nil {
		requestID = strings.TrimSpace(data.RequestID)
	}
	if requestID == "" {
		return nil, errors.New("缺少请求ID")
	}
	lastAlive := ctx.ConnInfo.LastAliveTime
	if lastAlive == 0 {
		lastAlive = ctx.ConnInfo.LastPingTime
	}
	if lastAlive < time.Now().UnixMilli()-int64(connectionMaxIdleSeconds*1000) {
		return nil, errors.New("当前连接已失活")
	}
	record, err := service.GetQuickLoginRequest(requestID, time.Now())
	if err != nil {
		return nil, err
	}
	if record == nil || strings.TrimSpace(record.TargetUserID) != strings.TrimSpace(ctx.User.ID) {
		return nil, errors.New("无权确认该快捷登录请求")
	}
	updated, err := service.ApproveQuickLoginRequest(service.ApproveQuickLoginRequestInput{
		RequestID:    requestID,
		ApproverID:   strings.TrimSpace(ctx.User.ID),
		ConnectionID: strings.TrimSpace(ctx.ConnInfo.ClientAddr) + ":" + strings.TrimSpace(ctx.User.ID),
	}, time.Now())
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"ok":        true,
		"requestId": updated.RequestID,
		"status":    updated.Status,
	}, nil
}

func apiQuickLoginDeny(ctx *ChatContext, data *quickLoginDenyPayload) (any, error) {
	if ctx == nil || ctx.User == nil || ctx.ConnInfo == nil {
		return nil, errors.New("未登录")
	}
	requestID := ""
	if data != nil {
		requestID = strings.TrimSpace(data.RequestID)
	}
	if requestID == "" {
		return nil, errors.New("缺少请求ID")
	}
	record, err := service.GetQuickLoginRequest(requestID, time.Now())
	if err != nil {
		return nil, err
	}
	if record == nil || strings.TrimSpace(record.TargetUserID) != strings.TrimSpace(ctx.User.ID) {
		return nil, errors.New("无权拒绝该快捷登录请求")
	}
	updated, err := service.DenyQuickLoginRequest(service.DenyQuickLoginRequestInput{
		RequestID: requestID,
	}, time.Now())
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"ok":        true,
		"requestId": updated.RequestID,
		"status":    updated.Status,
	}, nil
}

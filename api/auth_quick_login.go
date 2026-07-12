package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"sealchat/model"
	"sealchat/protocol"
	"sealchat/service"
	"sealchat/utils"
)

const defaultQuickLoginHint = "若该账号已在其他端在线，可向其发起一次快捷登录确认"

type quickLoginCheckRequest struct {
	Account string `json:"account"`
}

type quickLoginRequestBody struct {
	Account          string `json:"account"`
	CaptchaId        string `json:"captchaId"`
	CaptchaValue     string `json:"captchaValue"`
	TurnstileToken   string `json:"turnstileToken"`
	CapToken         string `json:"capToken"`
	RequesterBrowser string `json:"requesterBrowser"`
	RequesterDevice  string `json:"requesterDevice"`
}

type quickLoginPollRequest struct {
	RequestID      string `json:"requestId"`
	RequesterToken string `json:"requesterToken"`
}

func collectQuickLoginApproverConnections(
	userID string,
	userConnMap *utils.SyncMap[string, *utils.SyncMap[*WsSyncConn, *ConnInfo]],
	nowMs int64,
) []*ConnInfo {
	userID = strings.TrimSpace(userID)
	if userID == "" || userConnMap == nil {
		return nil
	}
	connMap, ok := userConnMap.Load(userID)
	if !ok || connMap == nil {
		return nil
	}
	cutoff := nowMs - int64(connectionMaxIdleSeconds*1000)
	items := make([]*ConnInfo, 0, 2)
	connMap.Range(func(_ *WsSyncConn, info *ConnInfo) bool {
		if info == nil {
			return true
		}
		lastAlive := info.LastAliveTime
		if lastAlive == 0 {
			lastAlive = info.LastPingTime
		}
		if lastAlive < cutoff {
			return true
		}
		items = append(items, info)
		return true
	})
	return items
}

func broadcastQuickLoginRequestEvent(
	targetUserID string,
	approvers []*ConnInfo,
	grant *service.QuickLoginGrant,
	accountInput, requesterIP, requesterBrowser, requesterDevice string,
) {
	targetUserID = strings.TrimSpace(targetUserID)
	if targetUserID == "" || grant == nil || len(approvers) == 0 {
		return
	}
	event := protocol.Event{
		Type: protocol.EventQuickLoginRequested,
		QuickLoginRequested: &protocol.QuickLoginRequestedPayload{
			RequestID:        grant.RequestID,
			AccountInput:     strings.TrimSpace(accountInput),
			RequestedAt:      time.Now().UnixMilli(),
			ExpiresAt:        grant.ExpiresAt.UnixMilli(),
			RequesterIP:      strings.TrimSpace(requesterIP),
			RequesterBrowser: strings.TrimSpace(requesterBrowser),
			RequesterDevice:  strings.TrimSpace(requesterDevice),
		},
	}
	payload := struct {
		protocol.Event
		Op protocol.Opcode `json:"op"`
	}{
		Event: event,
		Op:    protocol.OpEvent,
	}

	connMap := getUserConnInfoMap()
	if connMap == nil {
		return
	}
	userConnMap, ok := connMap.Load(targetUserID)
	if !ok || userConnMap == nil {
		return
	}
	userConnMap.Range(func(conn *WsSyncConn, info *ConnInfo) bool {
		if info == nil {
			return true
		}
		lastAlive := info.LastAliveTime
		if lastAlive == 0 {
			lastAlive = info.LastPingTime
		}
		if lastAlive < time.Now().UnixMilli()-int64(connectionMaxIdleSeconds*1000) {
			return true
		}
		writeConnJSONAndPrune(userConnMap, conn, payload)
		return true
	})
}

func QuickLoginCheck(c *fiber.Ctx) error {
	var body quickLoginCheckRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "请求参数错误"})
	}
	account := strings.TrimSpace(body.Account)
	if account == "" {
		return c.JSON(fiber.Map{
			"showQuickLoginButton": false,
			"hint":                 defaultQuickLoginHint,
		})
	}

	resolution, approvers, msg, err := resolveQuickLoginAvailability(account, time.Now())
	if err != nil {
		return c.JSON(fiber.Map{
			"showQuickLoginButton": false,
			"hint":                 "快捷登录状态查询失败，请稍后重试",
		})
	}
	if msg != "" || resolution == nil || resolution.User == nil || len(approvers) == 0 {
		if msg == "" {
			msg = defaultQuickLoginHint
		}
		return c.JSON(fiber.Map{
			"showQuickLoginButton": false,
			"hint":                 msg,
		})
	}
	return c.JSON(fiber.Map{
		"showQuickLoginButton": true,
		"hint":                 defaultQuickLoginHint,
	})
}

func QuickLoginRequest(c *fiber.Ctx) error {
	var body quickLoginRequestBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "请求参数错误"})
	}
	account := strings.TrimSpace(body.Account)
	if account == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "请输入用户名/昵称/邮箱"})
	}

	clientIP := getClientIP(c)
	resolution, approvers, msg, err := resolveQuickLoginAvailability(account, time.Now())
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "快捷登录查询失败，请稍后重试"})
	}
	if msg != "" || resolution == nil || resolution.User == nil || len(approvers) == 0 {
		if msg == "" {
			msg = "快捷登录暂不可用，请改用密码登录"
		}
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": msg})
	}
	if allowed, retryAfter := allowQuickLoginRequest(clientIP, resolution.User.ID, account, time.Now(), appConfig); !allowed {
		seconds := int(retryAfter.Round(time.Second) / time.Second)
		if seconds < 1 {
			seconds = 1
		}
		return c.Status(http.StatusTooManyRequests).JSON(fiber.Map{"message": fmt.Sprintf("快捷登录请求过于频繁，请在 %d 秒后重试", seconds)})
	}

	grant, err := service.CreateQuickLoginRequest(service.CreateQuickLoginRequestInput{
		TargetUserID:      resolution.User.ID,
		AccountInput:      account,
		RequesterIP:       clientIP,
		RequesterUA:       c.Get("User-Agent"),
		RequesterBrowser:  strings.TrimSpace(body.RequesterBrowser),
		RequesterDevice:   strings.TrimSpace(body.RequesterDevice),
		AvailableApprover: len(approvers),
	}, time.Now(), service.DefaultQuickLoginTTL)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "快捷登录发起失败，请稍后重试"})
	}

	broadcastQuickLoginRequestEvent(resolution.User.ID, approvers, grant, account, clientIP, body.RequesterBrowser, body.RequesterDevice)
	return c.JSON(fiber.Map{
		"requestId":      grant.RequestID,
		"requesterToken": grant.RequesterToken,
		"expiresAt":      grant.ExpiresAt.UnixMilli(),
	})
}

func QuickLoginPoll(c *fiber.Ctx) error {
	var body quickLoginPollRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "请求参数错误"})
	}

	poll, err := service.PollQuickLoginRequest(service.PollQuickLoginRequestInput{
		RequestID:      strings.TrimSpace(body.RequestID),
		RequesterToken: strings.TrimSpace(body.RequesterToken),
		RequesterIP:    getClientIP(c),
		RequesterUA:    c.Get("User-Agent"),
	}, time.Now())
	if err != nil {
		return writeQuickLoginPollError(c, err)
	}
	if poll.Status == service.QuickLoginStatusConsumed {
		return c.JSON(fiber.Map{"status": poll.Status, "token": poll.Token})
	}
	if poll.Status != service.QuickLoginStatusApproved {
		return c.JSON(fiber.Map{"status": poll.Status})
	}

	consumed, err := service.ConsumeQuickLoginRequest(service.ConsumeQuickLoginRequestInput{
		RequestID:      strings.TrimSpace(body.RequestID),
		RequesterToken: strings.TrimSpace(body.RequesterToken),
		RequesterIP:    getClientIP(c),
		RequesterUA:    c.Get("User-Agent"),
	}, time.Now())
	if err != nil {
		if errors.Is(err, service.ErrQuickLoginRequestNotFound) ||
			errors.Is(err, service.ErrQuickLoginRequesterTokenMismatch) ||
			errors.Is(err, service.ErrQuickLoginRequesterFingerprintMismatch) {
			return writeQuickLoginPollError(c, err)
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "快捷登录完成失败，请稍后重试"})
	}
	return c.JSON(fiber.Map{
		"status": consumed.Status,
		"token":  consumed.Token,
	})
}

func resolveQuickLoginAvailability(account string, now time.Time) (*model.SigninAccountResolution, []*ConnInfo, string, error) {
	resolution, err := model.ResolveUserBySigninAccount(account)
	if err != nil {
		return nil, nil, "", err
	}
	if resolution == nil {
		return nil, nil, "账号不存在，无法使用快捷登录", nil
	}
	switch resolution.Status {
	case model.SigninAccountStatusNotFound:
		return resolution, nil, "账号不存在，无法使用快捷登录", nil
	case model.SigninAccountStatusNicknameAmbiguous:
		return resolution, nil, "该昵称对应多个账号，请使用用户名或邮箱", nil
	case model.SigninAccountStatusMatched:
		if resolution.User == nil {
			return resolution, nil, "账号不存在，无法使用快捷登录", nil
		}
		approvers := collectQuickLoginApproverConnections(resolution.User.ID, getUserConnInfoMap(), now.UnixMilli())
		if len(approvers) == 0 {
			return resolution, nil, "该账号当前没有在线确认端，请改用密码登录", nil
		}
		return resolution, approvers, "", nil
	default:
		return resolution, nil, "快捷登录暂不可用，请改用密码登录", nil
	}
}

func writeQuickLoginPollError(c *fiber.Ctx, err error) error {
	if c == nil {
		return err
	}
	switch {
	case errors.Is(err, service.ErrQuickLoginRequesterFingerprintMismatch):
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "当前设备或网络环境已变更，请重新发起快捷登录"})
	case errors.Is(err, service.ErrQuickLoginRequesterTokenMismatch), errors.Is(err, service.ErrQuickLoginRequestNotFound):
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "快捷登录请求不存在或已失效"})
	default:
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "快捷登录已失效"})
	}
}

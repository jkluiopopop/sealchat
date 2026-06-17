package service

import (
	"fmt"
	"strings"
	"time"

	"sealchat/model"
)

func buildBattleReportSummaryPrompt(report *model.BattleReportModel, contextReports []*model.BattleReportModel, messages []*model.MessageModel) string {
	var builder strings.Builder
	builder.WriteString("请根据以下跑团聊天记录生成战报总结。\n")
	if report != nil {
		if !report.PeriodStart.IsZero() || !report.PeriodEnd.IsZero() {
			builder.WriteString("\n时间周期：")
			builder.WriteString(formatBattleReportTime(report.PeriodStart))
			builder.WriteString(" - ")
			builder.WriteString(formatBattleReportTime(report.PeriodEnd))
			builder.WriteString("\n")
		}
	}
	if len(contextReports) > 0 {
		builder.WriteString("\n前情提要：\n")
		for _, item := range contextReports {
			if item == nil {
				continue
			}
			builder.WriteString("## ")
			builder.WriteString(strings.TrimSpace(item.Title))
			builder.WriteString("\n")
			builder.WriteString(strings.TrimSpace(item.Content))
			builder.WriteString("\n")
		}
	}
	builder.WriteString("\n本次记录：\n")
	for _, msg := range messages {
		line := formatBattleReportMessageLine(msg)
		if line == "" {
			continue
		}
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	builder.WriteString("\n要求：忠实原意，按事件顺序整理，不要编造未出现的信息。")
	return strings.TrimSpace(builder.String())
}

func formatBattleReportMessageLine(msg *model.MessageModel) string {
	if msg == nil {
		return ""
	}
	content := strings.TrimSpace(buildFilteredPlainContent(msg.Content, false))
	if content == "" {
		return ""
	}
	name := strings.TrimSpace(msg.SenderIdentityName)
	if name == "" {
		name = strings.TrimSpace(msg.SenderMemberName)
	}
	if name == "" && msg.Member != nil {
		name = strings.TrimSpace(msg.Member.Nickname)
	}
	if name == "" && msg.User != nil {
		name = strings.TrimSpace(msg.User.Nickname)
		if name == "" {
			name = strings.TrimSpace(msg.User.Username)
		}
	}
	if name == "" {
		name = "未知发言者"
	}
	return fmt.Sprintf("[%s] %s：%s", formatBattleReportTime(msg.CreatedAt), name, content)
}

func formatBattleReportTime(value time.Time) string {
	if value.IsZero() {
		return "未知时间"
	}
	return value.Format("2006-01-02 15:04")
}

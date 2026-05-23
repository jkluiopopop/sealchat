package api

import (
	"encoding/json"
	"strings"

	"sealchat/service"
)

type tiptapMentionNode struct {
	Type    string              `json:"type"`
	Text    string              `json:"text"`
	Attrs   map[string]any      `json:"attrs"`
	Content []tiptapMentionNode `json:"content"`
}

func collectMentionTargetIDsFromContent(content string) map[string]struct{} {
	targets := make(map[string]struct{})
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return targets
	}

	for _, match := range atTagIDPattern.FindAllStringSubmatch(trimmed, -1) {
		id := strings.TrimSpace(firstNonEmptySubmatch(match, 1, 2))
		if id != "" {
			targets[id] = struct{}{}
		}
	}

	if service.LooksLikeTipTapJSON(trimmed) {
		var node tiptapMentionNode
		if err := json.Unmarshal([]byte(trimmed), &node); err == nil {
			collectMentionTargetIDsFromTipTapNode(&node, targets)
		}
	}

	return targets
}

func collectMentionTargetIDsFromTipTapNode(node *tiptapMentionNode, targets map[string]struct{}) {
	if node == nil {
		return
	}

	if text := strings.TrimSpace(node.Text); text != "" {
		for _, match := range atTagIDPattern.FindAllStringSubmatch(text, -1) {
			id := strings.TrimSpace(firstNonEmptySubmatch(match, 1, 2))
			if id != "" {
				targets[id] = struct{}{}
			}
		}
	}

	switch strings.ToLower(strings.TrimSpace(node.Type)) {
	case "mention", "satorimention":
		id := strings.TrimSpace(stringAttr(node.Attrs, "id"))
		if id != "" {
			targets[id] = struct{}{}
		}
	}

	for i := range node.Content {
		collectMentionTargetIDsFromTipTapNode(&node.Content[i], targets)
	}
}

func buildMessageCreatedNoticePayload(channelID, content, recipientID string) map[string]any {
	mentioned := false
	if recipientID != "" {
		targets := collectMentionTargetIDsFromContent(content)
		_, mentioned = targets[recipientID]
		if !mentioned {
			_, mentioned = targets["all"]
		}
	}
	return map[string]any{
		"op":        0,
		"type":      "message-created-notice",
		"channelId": channelID,
		"mentioned": mentioned,
	}
}

func firstNonEmptySubmatch(parts []string, indexes ...int) string {
	for _, index := range indexes {
		if index >= 0 && index < len(parts) {
			if trimmed := strings.TrimSpace(parts[index]); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func stringAttr(attrs map[string]any, key string) string {
	if len(attrs) == 0 {
		return ""
	}
	value, ok := attrs[key]
	if !ok {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	default:
		return ""
	}
}

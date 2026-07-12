package api

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"sealchat/model"
	"sealchat/protocol"
)

func isExternalBotIncomingUser(user *model.UserModel) bool {
	if user == nil || !user.IsBot {
		return false
	}
	switch strings.TrimSpace(user.BotKind) {
	case model.BotKindManual, model.BotKindChannelWebhook:
		return true
	default:
		return false
	}
}

func shouldTreatExternalBotMessageAsOOC(content string) bool {
	if first, ok := firstVisibleTextRune(content); ok {
		return first == '(' || first == '（'
	}
	trimmed := strings.TrimLeftFunc(content, unicode.IsSpace)
	return strings.HasPrefix(trimmed, "(") || strings.HasPrefix(trimmed, "（")
}

func resolveExternalBotIncomingICMode(requestedMode string, content string) string {
	if appConfig != nil && appConfig.BotIncomingParenAsOOC && shouldTreatExternalBotMessageAsOOC(content) {
		return "ooc"
	}
	requestedMode = strings.TrimSpace(strings.ToLower(requestedMode))
	return requestedMode
}

func firstVisibleTextRune(content string) (rune, bool) {
	root := protocol.ElementParse(content)
	if root == nil {
		return firstNonSpaceRune(content)
	}
	return firstVisibleTextRuneFromElement(root)
}

func firstVisibleTextRuneFromElement(el *protocol.Element) (rune, bool) {
	if el == nil {
		return 0, false
	}
	if el.Type == "text" {
		return firstNonSpaceRune(fmtSprintAttr(el.Attrs["content"]))
	}
	for _, child := range el.Children {
		if ch, ok := firstVisibleTextRuneFromElement(child); ok {
			return ch, true
		}
	}
	return 0, false
}

func firstNonSpaceRune(content string) (rune, bool) {
	trimmed := strings.TrimLeftFunc(content, unicode.IsSpace)
	if trimmed == "" {
		return 0, false
	}
	r, _ := utf8.DecodeRuneInString(trimmed)
	if r == utf8.RuneError {
		return 0, false
	}
	return r, true
}

func fmtSprintAttr(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}

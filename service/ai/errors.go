package ai

import (
	"errors"
	"fmt"
)

var ErrInputTooLong = errors.New("ai input too long")
var ErrUserCustomProviderRequired = errors.New("ai user custom provider required")

func FormatInputTooLongError(featureKey string, currentChars int, maxChars int) error {
	if featureKey == FeatureBattleSummary && currentChars > 0 && maxChars > 0 {
		return fmt.Errorf(
			"战报总结输入过长（当前 %d 字符，最大 %d 字符），请缩短时间范围或减少来源频道: %w",
			currentChars,
			maxChars,
			ErrInputTooLong,
		)
	}
	return ErrInputTooLong
}

func FormatUserCustomProviderRequiredError(featureKey string) error {
	label := BuiltinFeatures()[featureKey].Label
	if label == "" {
		label = "该 AI 功能"
	}
	return fmt.Errorf("%s仅允许用户自定义调用，请先在个人信息的 AI 设置中配置个人 API: %w", label, ErrUserCustomProviderRequired)
}

package ai

import (
	"errors"
	"fmt"
)

var ErrInputTooLong = errors.New("ai input too long")

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

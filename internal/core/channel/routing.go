package channel

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// StableRouteKey 将账号、聊天和线程组成不可逆的确定性键。
func StableRouteKey(accountID, chatID, threadID string) string {
	parts := []string{
		strings.TrimSpace(accountID),
		strings.TrimSpace(chatID),
		strings.TrimSpace(threadID),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])
}

// SplitText 按 Provider 限长切分 UTF-8 文本，优先在换行处断开。
func SplitText(value string, limit int) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if limit <= 0 {
		return []string{value}
	}
	runes := []rune(value)
	// 预分配切片容量，避免多次扩容。
	parts := make([]string, 0, (len(runes)+limit-1)/limit)
	for len(runes) > limit {
		cut := limit
		for i := limit - 1; i >= limit/2; i-- {
			if runes[i] == '\n' {
				cut = i + 1
				break
			}
		}
		part := strings.TrimSpace(string(runes[:cut]))
		if part != "" {
			parts = append(parts, part)
		}
		runes = runes[cut:]
	}
	if part := strings.TrimSpace(string(runes)); part != "" {
		parts = append(parts, part)
	}
	return parts
}

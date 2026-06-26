/**
 * @Time   : 2026/6/22 23:02
 * @Author : chenyangzhao542@gmail.com
 * @File   : chunker.go
 **/

package preprocessing

import (
	"regexp"
	"strings"

	"github.com/pkoukk/tiktoken-go"
)

// 记忆文本分块
// 把一段来源文件切成 512 token 的块逐块萃取
//
// 短文本整体作为一块；长文本按句子边界贪心聚合，块间不重叠
// 服用 RAG 的 tiktoken token 计数，保证与知识库分块口径一致

// MemoryChunkTokens 记忆分块目标 token 数
const MemoryChunkTokens int = 512

var sentSep = regexp.MustCompile(`[^。！？.!?\n]+[。！？.!?\n]?`)

type TextChunker struct {
	tkm *tiktoken.Tiktoken
}

func NewTextChunker() *TextChunker {
	tkm, _ := tiktoken.GetEncoding("cl100k_base")
	return &TextChunker{
		tkm: tkm,
	}
}

// Split 按句子聚合 MemoryChunkTokens 的块
// 短文本整体作为一块
func (c *TextChunker) Split(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{}
	}

	token := c.tkm.Encode(text, nil, nil)
	if len(token) <= MemoryChunkTokens {
		return []string{text}
	}

	// 按常见符号分割
	matches := sentSep.FindAllString(text, -1)
	parts := make([]string, 0, len(matches))
	for _, m := range matches {
		m = strings.TrimSpace(m)
		if m != "" {
			parts = append(parts, m)
		}
	}

	chunks := make([]string, 0)
	cur := make([]string, 0)
	curToken := 0
	for _, part := range parts {
		pt := len(c.tkm.Encode(part, nil, nil))
		if curToken+pt > MemoryChunkTokens && len(cur) > 0 {
			chunks = append(chunks, strings.Join(cur, ""))
			cur, curToken = make([]string, 0), 0
		}
		cur = append(cur, part)
		curToken += pt
	}

	if len(cur) > 0 {
		chunks = append(chunks, strings.Join(cur, ""))
	}

	return chunks
}

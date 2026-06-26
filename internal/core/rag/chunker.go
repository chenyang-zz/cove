package rag

import "strings"

type Chunk struct {
	ID       string
	Content  string
	Children []string
}

func ChunkText(text string, maxChars int) []Chunk {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if maxChars <= 0 {
		maxChars = 1200
	}
	runes := []rune(text)
	out := make([]Chunk, 0, len(runes)/maxChars+1)
	for start := 0; start < len(runes); start += maxChars {
		end := start + maxChars
		if end > len(runes) {
			end = len(runes)
		}
		content := string(runes[start:end])
		out = append(out, Chunk{ID: "", Content: content, Children: []string{content}})
	}
	return out
}

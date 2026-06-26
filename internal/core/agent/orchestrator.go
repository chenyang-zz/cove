package agent

import (
	"regexp"
	"strings"
)

type ReactAction struct {
	Tool  string
	Input string
}

var (
	actionRE      = regexp.MustCompile(`(?m)^Action\s*:\s*(.+)$`)
	actionInputRE = regexp.MustCompile(`(?m)^Action\s*Input\s*:\s*(.+)$`)
	finalRE       = regexp.MustCompile(`(?s)Final\s*Answer\s*:\s*(.*)$`)
)

func ParseReactAction(text string) (ReactAction, bool) {
	action := actionRE.FindStringSubmatch(text)
	if len(action) < 2 {
		return ReactAction{}, false
	}
	input := actionInputRE.FindStringSubmatch(text)
	value := ""
	if len(input) >= 2 {
		value = strings.TrimSpace(input[1])
	}
	return ReactAction{
		Tool:  strings.TrimSpace(action[1]),
		Input: value,
	}, true
}

func ParseReactFinal(text string) (string, bool) {
	match := finalRE.FindStringSubmatch(text)
	if len(match) < 2 {
		return "", false
	}
	return strings.TrimSpace(match[1]), true
}

package agent_test

import (
	"testing"

	"github.com/boxify/api-go/internal/core/agent"
)

func TestParseReactAction(t *testing.T) {
	action, ok := agent.ParseReactAction("Thought: need memory\nAction: memory_search\nAction Input: user preference")
	if !ok {
		t.Fatal("expected action to parse")
	}
	if action.Tool != "memory_search" {
		t.Fatalf("tool = %q", action.Tool)
	}
	if action.Input != "user preference" {
		t.Fatalf("input = %q", action.Input)
	}
}

func TestParseReactFinalAnswer(t *testing.T) {
	final, ok := agent.ParseReactFinal("Thought: enough\nFinal Answer: hello world")
	if !ok {
		t.Fatal("expected final answer")
	}
	if final != "hello world" {
		t.Fatalf("final = %q", final)
	}
}

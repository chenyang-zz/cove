package queue_test

import (
	"testing"

	"github.com/boxify/api-go/internal/infrastructure/queue"
)

func TestTaskNamesAreStable(t *testing.T) {
	names := queue.TaskNames()
	want := []string{
		"parse:document",
		"parse:image",
		"memory:extract",
		"memory:consolidate",
		"research:run",
	}
	if len(names) != len(want) {
		t.Fatalf("names = %#v", names)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("names[%d] = %q, want %q", i, names[i], want[i])
		}
	}
}

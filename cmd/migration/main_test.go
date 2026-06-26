package main

import (
	"context"
	"errors"
	"testing"
)

func TestRunCommandCallsMigration(t *testing.T) {
	runner := &fakeRunner{}
	if err := runCommand(context.Background(), runner); err != nil {
		t.Fatalf("runCommand error = %v", err)
	}
	if !runner.called {
		t.Fatal("runner was not called")
	}
}

func TestRunCommandReturnsRunnerError(t *testing.T) {
	want := errors.New("boom")
	runner := &fakeRunner{err: want}
	if err := runCommand(context.Background(), runner); !errors.Is(err, want) {
		t.Fatalf("runCommand error = %v, want %v", err, want)
	}
}

type fakeRunner struct {
	called bool
	err    error
}

func (r *fakeRunner) Up(context.Context) error {
	r.called = true
	return r.err
}

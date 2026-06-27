package storage_test

import (
	"context"
	"testing"

	"github.com/boxify/api-go/internal/infrastructure/storage"
)

func TestLocalStorePing(t *testing.T) {
	var store storage.Store = storage.NewLocalStore(t.TempDir())

	if err := store.Ping(context.Background()); err != nil {
		t.Fatalf("Ping error = %v", err)
	}
}

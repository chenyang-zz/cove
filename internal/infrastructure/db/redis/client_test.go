package redis_test

import (
	"context"
	"testing"

	infraredis "github.com/boxify/api-go/internal/infrastructure/db/redis"
)

func TestNewClientMapsConfigToOptions(t *testing.T) {
	client, err := infraredis.NewClient(context.Background(), infraredis.Config{
		Addr:     "redis:6379",
		Username: "user",
		Password: "password",
		DB:       2,
	})
	if err != nil {
		t.Fatalf("NewClient error = %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			t.Fatalf("Close error = %v", err)
		}
	}()

	options := client.Raw().Options()
	if options.Addr != "redis:6379" || options.Username != "user" || options.Password != "password" || options.DB != 2 {
		t.Fatalf("redis options = %#v", options)
	}
}

func TestNewClientRequiresAddr(t *testing.T) {
	if _, err := infraredis.NewClient(context.Background(), infraredis.Config{}); err == nil {
		t.Fatal("NewClient error = nil, want addr error")
	}
}

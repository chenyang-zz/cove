package graph

import (
	"context"
	"os"
	"testing"

	dbneo4j "github.com/boxify/api-go/internal/infrastructure/db/neo4j"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

func TestMemoryExampleRepositoryIntegrationWhenNeo4jEnvIsConfigured(t *testing.T) {
	uri := os.Getenv("NEO4J_TEST_URI")
	username := os.Getenv("NEO4J_TEST_USERNAME")
	password := os.Getenv("NEO4J_TEST_PASSWORD")
	if uri == "" || username == "" || password == "" {
		t.Skip("NEO4J_TEST_URI, NEO4J_TEST_USERNAME, and NEO4J_TEST_PASSWORD are required")
	}

	ctx := context.Background()
	client, err := dbneo4j.NewClient(ctx, dbneo4j.Config{
		URI:      uri,
		Username: username,
		Password: password,
		Database: os.Getenv("NEO4J_TEST_DATABASE"),
	})
	if err != nil {
		t.Fatalf("NewClient error = %v", err)
	}
	defer func() {
		if err := client.Close(ctx); err != nil {
			t.Fatalf("Close error = %v", err)
		}
	}()

	repo := NewMemoryExampleRepository(client)
	item := repository.MemoryExample{
		ID:     "memory-example-" + uuid.NewString(),
		UserID: "user-" + uuid.NewString(),
		Text:   "Neo4j repository example",
	}
	defer func() {
		_, _ = client.Write(ctx, "MATCH (m:MemoryExample {user_id: $user_id, id: $id}) DETACH DELETE m RETURN count(m) AS deleted", map[string]any{
			"user_id": item.UserID,
			"id":      item.ID,
		})
	}()

	created, err := repo.Upsert(ctx, item)
	if err != nil {
		t.Fatalf("Upsert error = %v", err)
	}
	if created != item {
		t.Fatalf("created = %#v, want %#v", created, item)
	}

	found, err := repo.FindByID(ctx, item.UserID, item.ID)
	if err != nil {
		t.Fatalf("FindByID error = %v", err)
	}
	if found != item {
		t.Fatalf("found = %#v, want %#v", found, item)
	}

	if err := repo.Delete(ctx, item.UserID, item.ID); err != nil {
		t.Fatalf("Delete error = %v", err)
	}

	if _, err := repo.FindByID(ctx, item.UserID, item.ID); xerr.From(err).Kind != xerr.KindNotFound {
		t.Fatalf("FindByID after delete error = %v, want not found", err)
	}
}

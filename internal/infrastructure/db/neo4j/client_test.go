package neo4j

import (
	"context"
	"os"
	"testing"

	"github.com/boxify/api-go/internal/xerr"
	neo4jdriver "github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

func TestNormalizeParamsReturnsEmptyMapForNil(t *testing.T) {
	got := normalizeParams(nil)
	if got == nil {
		t.Fatal("normalizeParams returned nil")
	}
	if len(got) != 0 {
		t.Fatalf("normalizeParams length = %d, want 0", len(got))
	}
}

func TestRowsFromRecordsPreservesKeysAndValues(t *testing.T) {
	records := []*neo4jdriver.Record{
		{Keys: []string{"name", "age"}, Values: []any{"neo", int64(5)}},
	}

	rows := rowsFromRecords(records)
	if len(rows) != 1 {
		t.Fatalf("rows len = %d, want 1", len(rows))
	}
	if rows[0]["name"] != "neo" || rows[0]["age"] != int64(5) {
		t.Fatalf("row = %#v", rows[0])
	}
}

func TestNormalizeDatabaseTrimsAndAllowsDefault(t *testing.T) {
	if got := normalizeDatabase("   "); got != "" {
		t.Fatalf("normalizeDatabase empty = %q, want empty", got)
	}
	if got := normalizeDatabase("  neo4j  "); got != "neo4j" {
		t.Fatalf("normalizeDatabase = %q, want neo4j", got)
	}
}

func TestPingRejectsUninitializedClient(t *testing.T) {
	err := (&Client{}).Ping(context.Background())
	if err == nil {
		t.Fatal("Ping returned nil error")
	}
	if xerr.From(err).Kind != xerr.KindBadRequest {
		t.Fatalf("error kind = %s, want %s", xerr.From(err).Kind, xerr.KindBadRequest)
	}
}

func TestClientIntegrationWhenNeo4jEnvIsConfigured(t *testing.T) {
	uri := os.Getenv("NEO4J_TEST_URI")
	username := os.Getenv("NEO4J_TEST_USERNAME")
	password := os.Getenv("NEO4J_TEST_PASSWORD")
	if uri == "" || username == "" || password == "" {
		t.Skip("NEO4J_TEST_URI, NEO4J_TEST_USERNAME, and NEO4J_TEST_PASSWORD are required")
	}

	ctx := context.Background()
	client, err := NewClient(ctx, Config{
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

	if err := client.Verify(ctx); err != nil {
		t.Fatalf("Verify error = %v", err)
	}
	if err := client.Ping(ctx); err != nil {
		t.Fatalf("Ping error = %v", err)
	}

	writeRows, err := client.Write(ctx, "CREATE (n:CodexNeo4jClientTest {id: $id}) RETURN n.id AS id", map[string]any{"id": "neo4j-client-test"})
	if err != nil {
		t.Fatalf("Write error = %v", err)
	}
	if len(writeRows) != 1 || writeRows[0]["id"] != "neo4j-client-test" {
		t.Fatalf("write rows = %#v", writeRows)
	}

	readRows, err := client.Read(ctx, "MATCH (n:CodexNeo4jClientTest {id: $id}) RETURN n.id AS id", map[string]any{"id": "neo4j-client-test"})
	if err != nil {
		t.Fatalf("Read error = %v", err)
	}
	if len(readRows) == 0 || readRows[0]["id"] != "neo4j-client-test" {
		t.Fatalf("read rows = %#v", readRows)
	}

	if _, err := client.Write(ctx, "MATCH (n:CodexNeo4jClientTest {id: $id}) DELETE n RETURN count(n) AS deleted", map[string]any{"id": "neo4j-client-test"}); err != nil {
		t.Fatalf("cleanup error = %v", err)
	}
}

func TestClientTxIntegrationWhenNeo4jEnvIsConfigured(t *testing.T) {
	uri := os.Getenv("NEO4J_TEST_URI")
	username := os.Getenv("NEO4J_TEST_USERNAME")
	password := os.Getenv("NEO4J_TEST_PASSWORD")
	if uri == "" || username == "" || password == "" {
		t.Skip("NEO4J_TEST_URI, NEO4J_TEST_USERNAME, and NEO4J_TEST_PASSWORD are required")
	}

	ctx := context.Background()
	client, err := NewClient(ctx, Config{
		URI:      uri,
		Username: username,
		Password: password,
		Database: os.Getenv("NEO4J_TEST_DATABASE"),
	})
	if err != nil {
		t.Fatalf("NewClient error = %v", err)
	}
	defer func() {
		_, _ = client.Write(ctx, "MATCH (n:CodexNeo4jClientTxTest) DELETE n RETURN count(n) AS deleted", nil)
		if err := client.Close(ctx); err != nil {
			t.Fatalf("Close error = %v", err)
		}
	}()

	err = client.WriteTx(ctx, func(tx Tx) error {
		if _, err := tx.Run(ctx, "CREATE (n:CodexNeo4jClientTxTest {id: $id}) RETURN n.id AS id", map[string]any{"id": "tx-1"}); err != nil {
			return err
		}
		if _, err := tx.Run(ctx, "CREATE (n:CodexNeo4jClientTxTest {id: $id}) RETURN n.id AS id", map[string]any{"id": "tx-2"}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WriteTx error = %v", err)
	}

	var ids []string
	err = client.ReadTx(ctx, func(tx Tx) error {
		rows, err := tx.Run(ctx, "MATCH (n:CodexNeo4jClientTxTest) RETURN n.id AS id ORDER BY id", nil)
		if err != nil {
			return err
		}
		for _, row := range rows {
			ids = append(ids, row["id"].(string))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("ReadTx error = %v", err)
	}
	if len(ids) != 2 || ids[0] != "tx-1" || ids[1] != "tx-2" {
		t.Fatalf("ids = %#v, want tx-1/tx-2", ids)
	}
}

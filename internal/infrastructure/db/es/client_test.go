package es_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	infraes "github.com/boxify/api-go/internal/infrastructure/db/es"
)

func TestClientPingAndRequests(t *testing.T) {
	var gotAuth string
	var gotIndexBody map[string]any
	var gotSearchBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		gotAuth = r.Header.Get("Authorization")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/":
			_, _ = w.Write([]byte(`{"version":{"number":"8.0.0"}}`))
		case r.Method == http.MethodPut && r.URL.Path == "/docs/_doc/1":
			if err := json.NewDecoder(r.Body).Decode(&gotIndexBody); err != nil {
				t.Fatalf("decode index body: %v", err)
			}
			_, _ = w.Write([]byte(`{"result":"created"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/docs/_doc/1":
			_, _ = w.Write([]byte(`{"_id":"1","found":true}`))
		case r.Method == http.MethodPost && r.URL.Path == "/docs/_search":
			if err := json.NewDecoder(r.Body).Decode(&gotSearchBody); err != nil {
				t.Fatalf("decode search body: %v", err)
			}
			_, _ = w.Write([]byte(`{"hits":{"total":{"value":1}}}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/docs/_doc/1":
			_, _ = w.Write([]byte(`{"result":"deleted"}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := infraes.NewClient(infraes.Config{
		URL:      server.URL,
		Username: "elastic",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("NewClient error = %v", err)
	}

	ctx := context.Background()
	if err := client.Ping(ctx); err != nil {
		t.Fatalf("Ping error = %v", err)
	}
	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("elastic:secret"))
	if gotAuth != wantAuth {
		t.Fatalf("authorization = %q, want %q", gotAuth, wantAuth)
	}

	indexResp, err := client.Index(ctx, "docs", "1", map[string]any{"title": "hello"})
	if err != nil {
		t.Fatalf("Index error = %v", err)
	}
	if indexResp["result"] != "created" || gotIndexBody["title"] != "hello" {
		t.Fatalf("index resp/body = %#v %#v", indexResp, gotIndexBody)
	}

	getResp, err := client.Get(ctx, "docs", "1")
	if err != nil {
		t.Fatalf("Get error = %v", err)
	}
	if getResp["_id"] != "1" || getResp["found"] != true {
		t.Fatalf("get resp = %#v", getResp)
	}

	searchResp, err := client.Search(ctx, "docs", map[string]any{"query": map[string]any{"match_all": map[string]any{}}})
	if err != nil {
		t.Fatalf("Search error = %v", err)
	}
	if _, ok := searchResp["hits"].(map[string]any); !ok || gotSearchBody["query"] == nil {
		t.Fatalf("search resp/body = %#v %#v", searchResp, gotSearchBody)
	}

	deleteResp, err := client.Delete(ctx, "docs", "1")
	if err != nil {
		t.Fatalf("Delete error = %v", err)
	}
	if deleteResp["result"] != "deleted" {
		t.Fatalf("delete resp = %#v", deleteResp)
	}
}

func TestNewClientSupportsAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		if got := r.Header.Get("Authorization"); got != "APIKey key-123" {
			t.Fatalf("authorization = %q", got)
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client, err := infraes.NewClient(infraes.Config{URL: server.URL, APIKey: "key-123"})
	if err != nil {
		t.Fatalf("NewClient error = %v", err)
	}
	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("Ping error = %v", err)
	}
}

func TestNewClientRequiresURL(t *testing.T) {
	if _, err := infraes.NewClient(infraes.Config{}); err == nil || !strings.Contains(err.Error(), "Elasticsearch URL") {
		t.Fatalf("NewClient error = %v, want url error", err)
	}
}

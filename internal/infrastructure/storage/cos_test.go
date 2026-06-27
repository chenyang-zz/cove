package storage_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/boxify/api-go/internal/infrastructure/storage"
	"github.com/google/uuid"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestCOSStorePutGetDeleteAndURL(t *testing.T) {
	var requests []string
	store, err := storage.NewCOSStore(storage.COSConfig{
		BucketURL: "https://bucket-1250000000.cos.ap-guangzhou.myqcloud.com",
		SecretID:  "secret-id",
		SecretKey: "secret-key",
		BaseURL:   "https://cdn.example.com/base",
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			requests = append(requests, r.Method+" "+r.URL.Path)
			switch r.Method {
			case http.MethodHead:
				return response(200, ""), nil
			case http.MethodPut:
				data, _ := io.ReadAll(r.Body)
				if string(data) != "hello" {
					t.Fatalf("put body = %q", string(data))
				}
				resp := response(200, "")
				resp.Header.Set("x-cos-hash-crc64ecma", "11177612005948864433")
				return resp, nil
			case http.MethodGet:
				return response(200, "hello"), nil
			case http.MethodDelete:
				return response(204, ""), nil
			default:
				t.Fatalf("unexpected method %s", r.Method)
				return response(500, ""), nil
			}
		}),
	})
	if err != nil {
		t.Fatalf("NewCOSStore error = %v", err)
	}

	ctx := context.Background()
	if err := store.Ping(ctx); err != nil {
		t.Fatalf("Ping error = %v", err)
	}
	if err := store.Put(ctx, "docs/a.txt", []byte("hello")); err != nil {
		t.Fatalf("Put error = %v", err)
	}
	data, err := store.Get(ctx, "docs/a.txt")
	if err != nil {
		t.Fatalf("Get error = %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("Get data = %q", string(data))
	}
	if err := store.Delete(ctx, "docs/a.txt"); err != nil {
		t.Fatalf("Delete error = %v", err)
	}
	if got := store.URL("docs/a.txt"); got != "https://cdn.example.com/base/docs/a.txt" {
		t.Fatalf("URL = %q", got)
	}
	if strings.Join(requests, ",") != "HEAD /,PUT /docs/a.txt,GET /docs/a.txt,DELETE /docs/a.txt" {
		t.Fatalf("requests = %#v", requests)
	}
}

func TestNewCOSStoreRequiresConfig(t *testing.T) {
	if _, err := storage.NewCOSStore(storage.COSConfig{}); err == nil {
		t.Fatal("NewCOSStore error = nil, want config error")
	}
}

func TestBuildFileKey(t *testing.T) {
	tests := []struct {
		name     string
		userID   uuid.UUID
		category string
		fileID   uuid.UUID
		ext      string
		want     string
	}{
		{
			name:     "adds leading dot",
			userID:   uuid.MustParse("11111111-1111-4111-8111-111111111111"),
			category: "documents",
			fileID:   uuid.MustParse("22222222-2222-4222-8222-222222222222"),
			ext:      "pdf",
			want:     "11111111-1111-4111-8111-111111111111/documents/22222222-2222-4222-8222-222222222222.pdf",
		},
		{
			name:     "keeps leading dot",
			userID:   uuid.MustParse("11111111-1111-4111-8111-111111111111"),
			category: "images",
			fileID:   uuid.MustParse("22222222-2222-4222-8222-222222222222"),
			ext:      ".png",
			want:     "11111111-1111-4111-8111-111111111111/images/22222222-2222-4222-8222-222222222222.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := storage.BuildFileKey(tt.userID, tt.category, tt.fileID, tt.ext)
			if got != tt.want {
				t.Fatalf("BuildFileKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{},
	}
}

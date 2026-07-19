package realdb_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	dbpostgres "github.com/boxify/api-go/internal/infrastructure/db/postgres"
	"github.com/boxify/api-go/internal/models"
	"github.com/google/uuid"
)

type knowledgeBaseData struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type documentData struct {
	ID       uuid.UUID  `json:"id"`
	KBID     *uuid.UUID `json:"kb_id"`
	FileName string     `json:"file_name"`
	Status   string     `json:"status"`
	Progress float64    `json:"progress"`
	ChunkNum int64      `json:"chunk_num"`
	ErrorMsg *string    `json:"error_msg"`
}

type documentStatusData struct {
	Status   string  `json:"status"`
	Progress float64 `json:"progress"`
	ErrorMsg *string `json:"error_msg"`
}

type searchDocumentData struct {
	SourceID uuid.UUID  `json:"source_id"`
	KBID     *uuid.UUID `json:"kb_id"`
	Content  string     `json:"content"`
}

type searchDocumentListData struct {
	List []*searchDocumentData `json:"list"`
}

// TestDocumentUploadCompletesThroughWorker 验证公共上传 API 会经真实 Redis worker、确定性向量服务和 Elasticsearch 完成 Markdown 入库。
func TestDocumentUploadCompletesThroughWorker(t *testing.T) {
	apiURL := strings.TrimRight(os.Getenv("COVE_REAL_DB_API_URL"), "/")
	databaseURL := os.Getenv("COVE_REAL_DB_DATABASE_URL")
	providerURL := strings.TrimRight(os.Getenv("COVE_REAL_DB_LLM_URL"), "/")
	if apiURL == "" || databaseURL == "" || providerURL == "" {
		t.Skip("COVE_REAL_DB_API_URL, COVE_REAL_DB_DATABASE_URL, and COVE_REAL_DB_LLM_URL are required")
	}

	db, err := dbpostgres.NewGormDB(t.Context(), dbpostgres.Config{URL: databaseURL})
	if err != nil {
		t.Fatalf("NewGormDB error = %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("DB error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := sqlDB.Close(); closeErr != nil {
			t.Errorf("Close DB error = %v", closeErr)
		}
	})

	client := &http.Client{Timeout: 20 * time.Second}
	runID := os.Getenv("COVE_REAL_DB_RUN_ID")
	owner := registerUser(t, client, apiURL, testUsername("document-upload", runID))
	var knowledgeBaseID uuid.UUID
	var modelConfigID uuid.UUID
	var documentID uuid.UUID
	t.Cleanup(func() {
		deletePublicResource(t, client, apiURL, owner.AccessToken, "/api/document/"+documentID.String(), documentID != uuid.Nil)
		deletePublicResource(t, client, apiURL, owner.AccessToken, "/api/model-configs/"+modelConfigID.String(), modelConfigID != uuid.Nil)
		deletePublicResource(t, client, apiURL, owner.AccessToken, "/api/knowledge-base/"+knowledgeBaseID.String(), knowledgeBaseID != uuid.Nil)
		result := db.WithContext(context.Background()).Where("id = ?", owner.UserID).Delete(&models.User{})
		if result.Error != nil {
			t.Errorf("cleanup document upload user error = %v", result.Error)
		}
	})

	knowledgeBase := doJSON[knowledgeBaseData](
		t,
		client,
		http.MethodPost,
		apiURL+"/api/knowledge-base",
		owner.AccessToken,
		map[string]string{
			"name":        "Upload fixture " + uuid.NewString()[:8],
			"description": "Run-owned harmless Markdown ingestion fixture",
		},
		http.StatusOK,
	)
	knowledgeBaseID = knowledgeBase.Data.ID
	if knowledgeBase.Code != 0 || knowledgeBaseID == uuid.Nil {
		t.Fatalf("knowledge base response = %+v, want a created knowledge base", knowledgeBase)
	}

	configured := doJSON[modelConfigData](
		t,
		client,
		http.MethodPost,
		apiURL+"/api/model-configs/",
		owner.AccessToken,
		map[string]any{
			"type":       "embedding",
			"provider":   "openai",
			"name":       "Local deterministic embedding",
			"model_name": "cove-e2e-embedding",
			"api_key":    "cove-e2e-local-key",
			"base_url":   providerURL,
			"is_default": true,
		},
		http.StatusOK,
	)
	modelConfigID = configured.Data.ID
	if configured.Code != 0 || modelConfigID == uuid.Nil || configured.Data.Type != "embedding" || !configured.Data.IsDefault || configured.Data.BaseURL != providerURL {
		t.Fatalf("model config response = %+v, want a default local embedding provider", configured)
	}

	fixtureToken := "cove-upload-" + uuid.NewString()
	fixture, err := os.ReadFile(filepath.Join("testdata", "harmless.md"))
	if err != nil {
		t.Fatalf("ReadFile harmless Markdown fixture error = %v", err)
	}
	markdown := strings.ReplaceAll(string(fixture), "{{RUN_TOKEN}}", fixtureToken)
	uploaded := uploadMarkdown(t, client, apiURL, owner.AccessToken, knowledgeBaseID, "harmless-fixture.md", markdown)
	documentID = uploaded.Data.ID
	if uploaded.Code != 0 || documentID == uuid.Nil || uploaded.Data.KBID == nil || *uploaded.Data.KBID != knowledgeBaseID {
		t.Fatalf("upload response = %+v, want a pending document in knowledge base %s", uploaded, knowledgeBaseID)
	}

	finalStatus := waitForDocumentDone(t, client, apiURL, owner.AccessToken, documentID, documentUploadTimeout(t))
	if finalStatus.Progress != 1 || finalStatus.ErrorMsg != nil {
		t.Fatalf("terminal document status = %+v, want done progress 1 without error", finalStatus)
	}

	detail := doJSON[documentData](
		t,
		client,
		http.MethodGet,
		fmt.Sprintf("%s/api/document/%s", apiURL, documentID),
		owner.AccessToken,
		nil,
		http.StatusOK,
	)
	if detail.Code != 0 || detail.Data.Status != "done" || detail.Data.ChunkNum <= 0 || detail.Data.KBID == nil || *detail.Data.KBID != knowledgeBaseID {
		t.Fatalf("document detail = %+v, want done with chunk_num > 0 in knowledge base %s", detail, knowledgeBaseID)
	}

	var persisted models.Document
	if err := db.WithContext(t.Context()).Where("id = ? AND user_id = ?", documentID, owner.UserID).First(&persisted).Error; err != nil {
		t.Fatalf("query persisted document error = %v", err)
	}
	if persisted.Status != "done" || persisted.ChunkNum != detail.Data.ChunkNum || persisted.KBID == nil || *persisted.KBID != knowledgeBaseID {
		t.Fatalf("persisted document status=%q chunk_num=%d kb_id=%v, want done chunk_num=%d kb_id=%s", persisted.Status, persisted.ChunkNum, persisted.KBID, detail.Data.ChunkNum, knowledgeBaseID)
	}

	waitForDocumentSearchResult(t, client, apiURL, owner.AccessToken, documentID, knowledgeBaseID, fixtureToken, 15*time.Second)
}

func uploadMarkdown(
	t *testing.T,
	client *http.Client,
	apiURL string,
	accessToken string,
	knowledgeBaseID uuid.UUID,
	fileName string,
	content string,
) *envelope[documentData] {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	filePart, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		t.Fatalf("CreateFormFile error = %v", err)
	}
	if _, err := io.WriteString(filePart, content); err != nil {
		t.Fatalf("write multipart file error = %v", err)
	}
	if err := writer.WriteField("kb_id", knowledgeBaseID.String()); err != nil {
		t.Fatalf("WriteField kb_id error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer error = %v", err)
	}

	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, apiURL+"/api/document/upload", &body)
	if err != nil {
		t.Fatalf("NewRequestWithContext upload error = %v", err)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("upload document request error = %v", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("ReadAll upload response error = %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("upload document status = %d, want 200; response_bytes=%d", response.StatusCode, len(responseBody))
	}
	var decoded envelope[documentData]
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		t.Fatalf("Unmarshal upload response error = %v; response_bytes=%d", err, len(responseBody))
	}
	return &decoded
}

func waitForDocumentDone(
	t *testing.T,
	client *http.Client,
	apiURL string,
	accessToken string,
	documentID uuid.UUID,
	timeout time.Duration,
) *documentStatusData {
	t.Helper()
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	var last documentStatusData
	for {
		status := doJSON[documentStatusData](
			t,
			client,
			http.MethodGet,
			fmt.Sprintf("%s/api/document/%s/status", apiURL, documentID),
			accessToken,
			nil,
			http.StatusOK,
		)
		last = status.Data
		switch last.Status {
		case "done":
			return &last
		case "failed":
			t.Fatalf("document %s failed at progress %.2f: %v", documentID, last.Progress, last.ErrorMsg)
		}
		if time.Now().After(deadline) {
			t.Fatalf("document %s did not become done within %s; last status=%q progress=%.2f error=%v", documentID, timeout, last.Status, last.Progress, last.ErrorMsg)
		}
		select {
		case <-t.Context().Done():
			t.Fatalf("document %s polling canceled: %v", documentID, t.Context().Err())
		case <-ticker.C:
		}
	}
}

func documentUploadTimeout(t *testing.T) time.Duration {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv("COVE_REAL_DB_DOCUMENT_TIMEOUT"))
	if raw == "" {
		return 60 * time.Second
	}
	timeout, err := time.ParseDuration(raw)
	if err != nil || timeout <= 0 {
		t.Fatalf("COVE_REAL_DB_DOCUMENT_TIMEOUT = %q, want a positive duration", raw)
	}
	return timeout
}

func containsDocumentSearchResult(rows []*searchDocumentData, documentID uuid.UUID, knowledgeBaseID uuid.UUID, token string) bool {
	for _, row := range rows {
		if row != nil && row.SourceID == documentID && row.KBID != nil && *row.KBID == knowledgeBaseID && strings.Contains(row.Content, token) {
			return true
		}
	}
	return false
}

func waitForDocumentSearchResult(
	t *testing.T,
	client *http.Client,
	apiURL string,
	accessToken string,
	documentID uuid.UUID,
	knowledgeBaseID uuid.UUID,
	token string,
	timeout time.Duration,
) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	lastCount := 0
	for {
		searched := doJSON[searchDocumentListData](
			t,
			client,
			http.MethodPost,
			apiURL+"/api/document/search",
			accessToken,
			map[string]any{"query": token, "top_k": 5},
			http.StatusOK,
		)
		lastCount = len(searched.Data.List)
		if searched.Code == 0 && containsDocumentSearchResult(searched.Data.List, documentID, knowledgeBaseID, token) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("document %s did not become searchable within %s; last result count=%d", documentID, timeout, lastCount)
		}
		select {
		case <-t.Context().Done():
			t.Fatalf("document %s search polling canceled: %v", documentID, t.Context().Err())
		case <-ticker.C:
		}
	}
}

func deletePublicResource(t *testing.T, client *http.Client, apiURL string, accessToken string, path string, enabled bool) {
	t.Helper()
	if !enabled {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, apiURL+path, nil)
	if err != nil {
		t.Errorf("cleanup NewRequestWithContext(%s) error = %v", path, err)
		return
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response, err := client.Do(request)
	if err != nil {
		t.Errorf("cleanup DELETE %s error = %v", path, err)
		return
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Errorf("cleanup ReadAll(%s) error = %v", path, err)
		return
	}
	if response.StatusCode != http.StatusOK {
		t.Errorf("cleanup DELETE %s status = %d, want 200; response_bytes=%d", path, response.StatusCode, len(responseBody))
	}
}

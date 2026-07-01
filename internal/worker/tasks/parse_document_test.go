package tasks

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/boxify/api-go/internal/domain"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

type fakeDocumentRepository struct {
	rows map[uuid.UUID]*models.Document
}

func newFakeDocumentRepository(rows ...*models.Document) *fakeDocumentRepository {
	repo := &fakeDocumentRepository{rows: map[uuid.UUID]*models.Document{}}
	for _, row := range rows {
		repo.rows[row.ID] = row
	}
	return repo
}

func (r *fakeDocumentRepository) Create(ctx context.Context, userID uuid.UUID, document *models.Document) (*models.Document, error) {
	document.UserID = userID
	r.rows[document.ID] = document
	return document, nil
}

func (r *fakeDocumentRepository) List(ctx context.Context, userID uuid.UUID) ([]*models.Document, error) {
	return nil, nil
}

func (r *fakeDocumentRepository) PageList(ctx context.Context, userID uuid.UUID, query repository.DocumentListQuery) ([]*models.Document, int64, error) {
	return nil, 0, nil
}

func (r *fakeDocumentRepository) CountByKnowledgeBase(ctx context.Context, userID uuid.UUID, kbIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	return nil, nil
}

func (r *fakeDocumentRepository) FindByID(ctx context.Context, userID uuid.UUID, documentID uuid.UUID) (*models.Document, error) {
	row, ok := r.rows[documentID]
	if !ok || row.UserID != userID {
		return nil, xerr.NotFound("文档不存在")
	}
	return row, nil
}

func (r *fakeDocumentRepository) Update(ctx context.Context, userID uuid.UUID, document *models.Document) (*models.Document, error) {
	r.rows[document.ID] = document
	return document, nil
}

func (r *fakeDocumentRepository) UpdateFields(ctx context.Context, userID uuid.UUID, documentID uuid.UUID, document *models.Document, fields *repository.DocumentUpdateFields) (*models.Document, error) {
	row, err := r.FindByID(ctx, userID, documentID)
	if err != nil {
		return nil, err
	}
	for _, column := range fields.Columns() {
		switch column {
		case "status":
			row.Status = document.Status
		case "progress":
			row.Progress = document.Progress
		case "chunk_num":
			row.ChunkNum = document.ChunkNum
		case "error_msg":
			row.ErrorMsg = document.ErrorMsg
		}
	}
	return row, nil
}

func (r *fakeDocumentRepository) Delete(ctx context.Context, userID uuid.UUID, documentID uuid.UUID) error {
	delete(r.rows, documentID)
	return nil
}

type memoryStore struct {
	data map[string][]byte
}

func newMemoryStore() *memoryStore {
	return &memoryStore{data: map[string][]byte{}}
}

func (s *memoryStore) Ping(ctx context.Context) error {
	return nil
}

func (s *memoryStore) Put(ctx context.Context, key string, data []byte) error {
	s.data[key] = append([]byte(nil), data...)
	return nil
}

func (s *memoryStore) Get(ctx context.Context, key string) ([]byte, error) {
	data, ok := s.data[key]
	if !ok {
		return nil, xerr.NotFound("文件不存在")
	}
	return append([]byte(nil), data...), nil
}

func (s *memoryStore) Delete(ctx context.Context, key string) error {
	delete(s.data, key)
	return nil
}

func TestHandleParseDocumentProcessesTextDocument(t *testing.T) {
	// 验证 parse:document handler 会读取文本文件、完成分块，并把文档状态更新为 done。
	ctx := context.Background()
	userID := uuid.New()
	documentID := uuid.New()
	row := &models.Document{ID: documentID, UserID: userID, FileName: "a.txt", FileExt: ".txt", FileKey: "docs/a.txt", Status: "pending"}
	store := newMemoryStore()
	store.data[row.FileKey] = []byte("hello async queue")
	handler := NewParseDocumentTask(&svc.ServiceContext{DocumentRepo: newFakeDocumentRepository(row), Storage: store})
	task, err := domain.NewParseDocumentTask(userID, documentID)
	if err != nil {
		t.Fatalf("NewParseDocumentTask error = %v", err)
	}

	if err := handler.Handle(ctx, task); err != nil {
		t.Fatalf("HandleParseDocument error = %v", err)
	}
	if row.Status != "done" || row.Progress != 1 || row.ChunkNum != 1 || row.ErrorMsg != nil {
		t.Fatalf("document after parse = %+v, want done/progress/chunk/error cleared", row)
	}
}

func TestHandleParseDocumentMarksUnsupportedDocumentFailed(t *testing.T) {
	// 验证 PDF/DOCX 在 Go 解析器未接入前不会反复重试，而是写入 failed 和明确错误。
	ctx := context.Background()
	userID := uuid.New()
	documentID := uuid.New()
	row := &models.Document{ID: documentID, UserID: userID, FileName: "a.pdf", FileExt: ".pdf", FileKey: "docs/a.pdf", Status: "pending"}
	store := newMemoryStore()
	store.data[row.FileKey] = []byte("%PDF")
	handler := NewParseDocumentTask(&svc.ServiceContext{DocumentRepo: newFakeDocumentRepository(row), Storage: store})
	task, err := domain.NewParseDocumentTask(userID, documentID)
	if err != nil {
		t.Fatalf("NewParseDocumentTask error = %v", err)
	}

	if err := handler.Handle(ctx, task); err != nil {
		t.Fatalf("HandleParseDocument error = %v", err)
	}
	if row.Status != "failed" || row.ErrorMsg == nil || !strings.Contains(*row.ErrorMsg, "暂不支持") {
		t.Fatalf("document after unsupported parse = %+v, want failed unsupported parser message", row)
	}
}

func TestHandleParseDocumentSkipsRetryWhenDocumentMissing(t *testing.T) {
	// 验证任务中的文档已被删除时返回 SkipRetry，避免无意义重试。
	ctx := context.Background()
	userID := uuid.New()
	task, err := domain.NewParseDocumentTask(userID, uuid.New())
	if err != nil {
		t.Fatalf("NewParseDocumentTask error = %v", err)
	}
	handler := NewParseDocumentTask(&svc.ServiceContext{DocumentRepo: newFakeDocumentRepository(), Storage: newMemoryStore()})

	if err := handler.Handle(ctx, task); !errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("HandleParseDocument missing error = %v, want SkipRetry", err)
	}
}

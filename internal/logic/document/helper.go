package document

import (
	"context"
	"strings"

	"github.com/boxify/api-go/internal/domain"
	"github.com/boxify/api-go/internal/infrastructure/queue"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

const (
	documentSourceFile  = "file"
	maxDocumentFileSize = 50 * 1024 * 1024
	previewMaxChars     = 80000
)

var supportedDocumentExts = map[string]struct{}{
	".pdf":      {},
	".docx":     {},
	".md":       {},
	".markdown": {},
	".txt":      {},
	".html":     {},
	".htm":      {},
}

var previewTextExts = map[string]struct{}{
	".md":       {},
	".markdown": {},
	".txt":      {},
	".html":     {},
	".htm":      {},
}

func parseDocumentID(raw string) (uuid.UUID, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return uuid.Nil, xerr.BadRequest("文档 ID 无效")
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, xerr.BadRequest("文档 ID 无效")
	}
	return id, nil
}

func parseOptionalKBID(raw *string) (*uuid.UUID, error) {
	if raw == nil {
		return nil, nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil, nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return nil, xerr.BadRequest("知识库 ID 无效")
	}
	return &id, nil
}

func parseRequiredKBID(raw string) (uuid.UUID, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return uuid.Nil, xerr.BadRequest("知识库 ID 无效")
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, xerr.BadRequest("知识库 ID 无效")
	}
	return id, nil
}

func supportedDocumentExt(ext string) (string, error) {
	ext = strings.ToLower(ext)
	if _, ok := supportedDocumentExts[ext]; !ok {
		return "", xerr.BadRequestf("不支持的文件类型: %s", ext)
	}
	return ext, nil
}

func isPreviewTextExt(ext string) bool {
	_, ok := previewTextExts[strings.ToLower(ext)]
	return ok
}

func truncatePreview(text string) (string, bool) {
	runes := []rune(text)
	if len(runes) <= previewMaxChars {
		return text, false
	}
	return string(runes[:previewMaxChars]), true
}

func enqueueParseDocumentTask(ctx context.Context, producer queue.Producer, userID uuid.UUID, documentID uuid.UUID) error {
	if producer == nil {
		return xerr.Internal("任务队列未初始化", nil)
	}
	task, err := domain.NewParseDocumentTask(userID, documentID)
	if err != nil {
		return xerr.Wrapf(err, "创建文档解析任务失败")
	}
	_, err = producer.Enqueue(ctx, task)
	if err != nil {
		return xerr.Wrapf(err, "提交文档解析任务失败")
	}
	return nil
}

func markDocumentParseDispatchFailed(ctx context.Context, repo repository.DocumentRepository, userID uuid.UUID, documentID uuid.UUID, cause error) {
	if repo == nil || cause == nil {
		return
	}
	message := cause.Error()
	_, _ = repo.UpdateFields(ctx, userID, documentID, &models.Document{
		Status:   domain.DocumentStatusFailed,
		Progress: 0,
		ErrorMsg: &message,
	}, repository.NewDocumentUpdateFields().Status().Progress().ErrorMsg())
}

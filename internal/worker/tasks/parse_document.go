package tasks

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/boxify/api-go/internal/core/rag"
	"github.com/boxify/api-go/internal/domain"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

var errUnsupportedDocumentParser = errors.New("暂不支持该文件类型的后台解析")

type ParseDocumentTask struct {
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

func NewParseDocumentTask(svcCtx *svc.ServiceContext) *ParseDocumentTask {
	return &ParseDocumentTask{
		svcCtx: svcCtx,
		log:    xlog.Component("worker.tasks.parse_document"),
	}
}

func (h *ParseDocumentTask) Handle(ctx context.Context, task *domain.Task) error {
	payload, err := parseTaskPayload(task)
	if err != nil {
		return skipRetry(fmt.Errorf("解析文档任务 payload 失败: %w", err))
	}
	if h == nil || h.svcCtx == nil || h.svcCtx.DocumentRepo == nil || h.svcCtx.Storage == nil {
		return xerr.Internal("文档解析任务依赖未初始化", nil)
	}

	doc, err := h.svcCtx.DocumentRepo.FindByID(ctx, payload.UserID, payload.DocumentID)
	if err != nil {
		if xerr.From(err).Kind == xerr.KindNotFound {
			h.log.WarnContext(ctx, "文档不存在，跳过解析任务",
				slog.String("user_id", payload.UserID.String()),
				slog.String("document_id", payload.DocumentID.String()),
			)
			return skipRetry(err)
		}
		return err
	}

	h.log.InfoContext(ctx, "开始解析文档",
		slog.String("user_id", payload.UserID.String()),
		slog.String("document_id", payload.DocumentID.String()),
		slog.String("file_ext", doc.FileExt),
	)

	if err := h.updateParseState(ctx, doc, &models.Document{
		Status:   domain.DocumentStatusParsing,
		Progress: 0.1,
		ErrorMsg: nil,
	}, repository.NewDocumentUpdateFields().Status().Progress().ErrorMsg()); err != nil {
		return err
	}

	content, err := h.svcCtx.Storage.Get(ctx, doc.FileKey)
	if err != nil {
		_ = h.markParseFailed(ctx, doc, err)
		return err
	}
	text, err := parseDocumentText(doc.FileExt, content)
	if err != nil {
		_ = h.markParseFailed(ctx, doc, err)
		if errors.Is(err, errUnsupportedDocumentParser) {
			h.log.WarnContext(ctx, "文档解析器暂不支持该文件类型",
				slog.String("document_id", payload.DocumentID.String()),
				slog.String("file_ext", doc.FileExt),
			)
			return nil
		}
		return err
	}
	chunks := rag.ChunkText(text, 1200)
	if len(chunks) == 0 {
		err := errors.New("解析结果为空")
		_ = h.markParseFailed(ctx, doc, err)
		return nil
	}

	if err := h.updateParseState(ctx, doc, &models.Document{
		Status:   domain.DocumentStatusDone,
		Progress: 1,
		ChunkNum: int64(len(chunks)),
		ErrorMsg: nil,
	}, repository.NewDocumentUpdateFields().Status().Progress().ChunkNum().ErrorMsg()); err != nil {
		return err
	}
	h.log.InfoContext(ctx, "文档解析完成",
		slog.String("user_id", payload.UserID.String()),
		slog.String("document_id", payload.DocumentID.String()),
		slog.Int("chunk_count", len(chunks)),
	)
	return nil
}

func parseTaskPayload(task *domain.Task) (*domain.ParseDocumentPayload, error) {
	if task == nil {
		return nil, fmt.Errorf("task is nil")
	}
	payload, ok := task.Payload.(*domain.ParseDocumentPayload)
	if !ok || payload == nil {
		return nil, fmt.Errorf("payload type = %T", task.Payload)
	}
	if payload.UserID == uuid.Nil || payload.DocumentID == uuid.Nil {
		return nil, fmt.Errorf("payload ids are required")
	}
	return payload, nil
}

func (h *ParseDocumentTask) markParseFailed(ctx context.Context, doc *models.Document, cause error) error {
	message := cause.Error()
	if h != nil && h.log != nil {
		h.log.WarnContext(ctx, "文档解析失败",
			slog.String("document_id", doc.ID.String()),
			slog.String("error", message),
		)
	}
	return h.updateParseState(ctx, doc, &models.Document{
		Status:   domain.DocumentStatusFailed,
		Progress: doc.Progress,
		ErrorMsg: &message,
	}, repository.NewDocumentUpdateFields().Status().Progress().ErrorMsg())
}

func (h *ParseDocumentTask) updateParseState(ctx context.Context, doc *models.Document, patch *models.Document, fields *repository.DocumentUpdateFields) error {
	_, err := h.svcCtx.DocumentRepo.UpdateFields(ctx, doc.UserID, doc.ID, patch, fields)
	return err
}

func parseDocumentText(fileExt string, content []byte) (string, error) {
	switch strings.ToLower(strings.TrimSpace(fileExt)) {
	case ".md", ".markdown", ".txt", ".html", ".htm":
		text := strings.TrimSpace(string(content))
		if text == "" {
			return "", errors.New("解析结果为空")
		}
		return text, nil
	case ".pdf", ".docx":
		return "", fmt.Errorf("%w: %s", errUnsupportedDocumentParser, fileExt)
	default:
		return "", fmt.Errorf("%w: %s", errUnsupportedDocumentParser, fileExt)
	}
}

func skipRetry(err error) error {
	return errors.Join(err, asynq.SkipRetry)
}

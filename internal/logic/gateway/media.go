package gateway

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	corechannel "github.com/boxify/api-go/internal/core/channel"
	corellm "github.com/boxify/api-go/internal/core/llm"
	ragparser "github.com/boxify/api-go/internal/core/rag/documentparse"
	ragimagedescribe "github.com/boxify/api-go/internal/core/rag/imagedescribe"
	"github.com/boxify/api-go/internal/domain/types"
	"github.com/boxify/api-go/internal/infrastructure/storage"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/svc"
	"github.com/google/uuid"
)

const maxExtractedTextRunes = 200_000

// ProcessMedia 下载、校验并解析渠道附件。单个附件解析失败不会丢弃已保存原文件。
func (s *Service) ProcessMedia(ctx context.Context, account *models.ChannelAccount, media []corechannel.MediaReference) ([]models.MessageAttachmentMeta, []*types.MessageAttachment, error) {
	if len(media) == 0 {
		return nil, nil, nil
	}
	provider, ok := s.svc.ChannelRegistry.Get(corechannel.ProviderName(account.Provider))
	if !ok {
		return nil, nil, errors.New("channel provider is unavailable")
	}
	downloader, ok := provider.(corechannel.MediaDownloader)
	if !ok {
		return nil, nil, errors.New("channel provider does not support media download")
	}
	accountConfig, err := s.AccountConfig(account)
	if err != nil {
		return nil, nil, err
	}
	maxBytes := s.svc.Config.Gateway.MaxMediaBytes
	if maxBytes <= 0 {
		maxBytes = 20 << 20
	}
	metas := make([]models.MessageAttachmentMeta, 0, len(media))
	attachments := make([]*types.MessageAttachment, 0, len(media))
	for _, reference := range media {
		meta, attachment, processErr := s.processOneMedia(ctx, downloader, accountConfig, account.UserID, reference, maxBytes)
		if processErr != nil {
			return metas, attachments, processErr
		}
		metas = append(metas, meta)
		if attachment != nil && strings.TrimSpace(attachment.Content) != "" {
			attachments = append(attachments, attachment)
		}
	}
	return metas, attachments, nil
}

func (s *Service) processOneMedia(ctx context.Context, downloader corechannel.MediaDownloader, account corechannel.AccountConfig, userID uuid.UUID, reference corechannel.MediaReference, maxBytes int64) (models.MessageAttachmentMeta, *types.MessageAttachment, error) {
	if reference.Size > maxBytes {
		return models.MessageAttachmentMeta{}, nil, fmt.Errorf("附件超过 %d 字节限制", maxBytes)
	}
	downloaded, err := downloader.DownloadMedia(ctx, account, reference)
	if err != nil {
		return models.MessageAttachmentMeta{}, nil, fmt.Errorf("下载渠道附件失败: %w", err)
	}
	if downloaded == nil || downloaded.Body == nil {
		return models.MessageAttachmentMeta{}, nil, errors.New("渠道附件为空")
	}
	defer downloaded.Body.Close()
	if downloaded.Size > maxBytes {
		return models.MessageAttachmentMeta{}, nil, fmt.Errorf("附件超过 %d 字节限制", maxBytes)
	}
	data, err := io.ReadAll(io.LimitReader(downloaded.Body, maxBytes+1))
	if err != nil {
		return models.MessageAttachmentMeta{}, nil, err
	}
	if int64(len(data)) > maxBytes {
		return models.MessageAttachmentMeta{}, nil, fmt.Errorf("附件超过 %d 字节限制", maxBytes)
	}
	declaredMIME := cleanMIME(firstNonEmptyString(downloaded.MIMEType, reference.MIMEType))
	detectedMIME := cleanMIME(http.DetectContentType(data))
	if !allowedMedia(reference.Kind, declaredMIME, detectedMIME) {
		return models.MessageAttachmentMeta{}, nil, fmt.Errorf("不支持的附件类型 %s", firstNonEmptyString(declaredMIME, detectedMIME))
	}
	fileName := firstNonEmptyString(downloaded.FileName, reference.FileName, "attachment")
	ext := mediaExtension(fileName, declaredMIME, reference.Kind)
	fileID := uuid.New()
	key := storage.BuildFileKey(userID, "gateway", fileID, ext)
	if err := s.svc.Storage.Put(ctx, key, data); err != nil {
		return models.MessageAttachmentMeta{}, nil, err
	}
	meta := models.MessageAttachmentMeta{
		Kind: reference.Kind, FileName: fileName, MIMEType: firstNonEmptyString(declaredMIME, detectedMIME),
		StorageKey: key, Size: int64(len(data)),
	}
	attachment := &types.MessageAttachment{FileName: fileName}
	switch reference.Kind {
	case "image":
		attachment.Content, err = s.describeGatewayImage(ctx, userID, data, ext)
	case "document":
		var parsed *ragparser.Output
		if s.svc.RAGDocumentParser == nil {
			err = errors.New("document parser is unavailable")
		} else {
			parsed, err = s.svc.RAGDocumentParser.Parse(ctx, ragparser.Input{Data: data, FileExt: ext})
			if err == nil && parsed != nil {
				attachment.Content = parsed.Text
			}
		}
	default:
		err = errors.New("unsupported gateway media kind")
	}
	if err != nil {
		meta.ParseError = "附件内容解析失败"
		return meta, nil, nil
	}
	attachment.Content = truncateExtractedText(attachment.Content)
	meta.ExtractedText = attachment.Content
	return meta, attachment, nil
}

func (s *Service) describeGatewayImage(ctx context.Context, userID uuid.UUID, data []byte, ext string) (string, error) {
	client, err := svc.MultimodalClient(ctx, s.svc, userID)
	if err != nil {
		return "", err
	}
	vision, ok := client.(corellm.VisionClient)
	if !ok {
		return "", errors.New("multimodal client does not support vision")
	}
	description, err := ragimagedescribe.NewDescriber(vision).Describe(ctx, ragimagedescribe.Input{Data: data, FileExt: ext})
	if err != nil {
		return "", err
	}
	parts := []string{description.Description}
	if description.OCRText != "" {
		parts = append(parts, "OCR: "+description.OCRText)
	}
	if description.Scene != "" {
		parts = append(parts, "Scene: "+description.Scene)
	}
	return strings.Join(nonEmptyStrings(parts), "\n"), nil
}

func allowedMedia(kind, declared, detected string) bool {
	switch kind {
	case "image":
		return allowedImageMIME(firstNonEmptyString(declared, detected)) && allowedImageMIME(detected)
	case "document":
		declared = firstNonEmptyString(declared, detected)
		if !allowedDocumentMIME(declared) {
			return false
		}
		return allowedDocumentMIME(detected) || (declared == "application/vnd.openxmlformats-officedocument.wordprocessingml.document" && detected == "application/zip")
	default:
		return false
	}
}

func allowedImageMIME(value string) bool {
	switch value {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
		return true
	default:
		return false
	}
}

func allowedDocumentMIME(value string) bool {
	switch value {
	case "application/pdf", "text/plain", "text/markdown", "text/html", "application/xhtml+xml", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "application/zip":
		return true
	default:
		return false
	}
}

func mediaExtension(fileName, mimeType, kind string) string {
	if ext := strings.ToLower(filepath.Ext(filepath.Base(fileName))); ext != "" && len(ext) <= 8 {
		return ext
	}
	byMIME := map[string]string{
		"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp", "image/gif": ".gif",
		"application/pdf": ".pdf", "text/plain": ".txt", "text/markdown": ".md", "text/html": ".html",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": ".docx",
	}
	if ext := byMIME[mimeType]; ext != "" {
		return ext
	}
	if kind == "image" {
		return ".jpg"
	}
	return ".bin"
}

func cleanMIME(value string) string {
	return strings.ToLower(strings.TrimSpace(strings.Split(value, ";")[0]))
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func nonEmptyStrings(values []string) []string {
	out := values[:0]
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}

func truncateExtractedText(value string) string {
	runes := []rune(value)
	if len(runes) <= maxExtractedTextRunes {
		return value
	}
	return string(runes[:maxExtractedTextRunes]) + "\n[附件提取内容已截断]"
}

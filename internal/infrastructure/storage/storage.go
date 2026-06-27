package storage

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

type Store interface {
	Ping(ctx context.Context) error
	Put(ctx context.Context, key string, data []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}

type URLSigner interface {
	URL(key string) string
}

// BuildFileKey 生成对象存储文件 key：{user_id}/{category}/{file_id}.ext。
// category 通常为 documents、images 等业务目录。
func BuildFileKey(userID uuid.UUID, category string, fileID uuid.UUID, ext string) string {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return userID.String() + "/" + category + "/" + fileID.String() + ext
}

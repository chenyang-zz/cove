package storage

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	cos "github.com/tencentyun/cos-go-sdk-v5"

	"github.com/boxify/api-go/internal/xerr"
)

type COSConfig struct {
	BucketURL string
	SecretID  string
	SecretKey string
	BaseURL   string
	Transport http.RoundTripper
}

type COSStore struct {
	client  *cos.Client
	baseURL string
}

func NewCOSStore(cfg COSConfig) (*COSStore, error) {
	if cfg.BucketURL == "" || cfg.SecretID == "" || cfg.SecretKey == "" {
		return nil, xerr.BadRequest("COS 存储配置无效")
	}
	bucketURL, err := url.Parse(cfg.BucketURL)
	if err != nil {
		return nil, xerr.Wrapf(err, "解析 COS bucket_url 失败")
	}
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = strings.TrimRight(cfg.BucketURL, "/")
	}
	transport := cfg.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	client := cos.NewClient(&cos.BaseURL{BucketURL: bucketURL}, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  cfg.SecretID,
			SecretKey: cfg.SecretKey,
			Transport: transport,
		},
	})
	return &COSStore{client: client, baseURL: baseURL}, nil
}

func (s *COSStore) Put(ctx context.Context, key string, data []byte) error {
	if _, err := s.client.Object.Put(ctx, cleanObjectKey(key), bytes.NewReader(data), nil); err != nil {
		return xerr.Wrapf(err, "写入 COS 对象失败: key=%s", key)
	}
	return nil
}

func (s *COSStore) Get(ctx context.Context, key string) ([]byte, error) {
	resp, err := s.client.Object.Get(ctx, cleanObjectKey(key), nil)
	if err != nil {
		return nil, xerr.Wrapf(err, "读取 COS 对象失败: key=%s", key)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, xerr.Wrapf(err, "读取 COS 响应失败: key=%s", key)
	}
	return data, nil
}

func (s *COSStore) Delete(ctx context.Context, key string) error {
	if _, err := s.client.Object.Delete(ctx, cleanObjectKey(key)); err != nil {
		return xerr.Wrapf(err, "删除 COS 对象失败: key=%s", key)
	}
	return nil
}

func (s *COSStore) Ping(ctx context.Context) error {
	if s == nil || s.client == nil {
		return xerr.BadRequest("COS 存储未初始化")
	}
	if _, err := s.client.Bucket.Head(ctx); err != nil {
		return xerr.Wrapf(err, "验证 COS bucket 连接失败")
	}
	return nil
}

func (s *COSStore) URL(key string) string {
	return strings.TrimRight(s.baseURL, "/") + "/" + cleanObjectKey(key)
}

func cleanObjectKey(key string) string {
	key = strings.TrimPrefix(path.Clean("/"+key), "/")
	if key == "." {
		return ""
	}
	return key
}

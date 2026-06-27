package es

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"

	"github.com/boxify/api-go/internal/xerr"
)

type Config struct {
	URL      string
	Username string
	Password string
	APIKey   string
}

type Client struct {
	raw *elasticsearch.Client
}

func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, xerr.BadRequest("Elasticsearch URL 配置无效")
	}
	raw, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{cfg.URL},
		Username:  cfg.Username,
		Password:  cfg.Password,
		APIKey:    cfg.APIKey,
	})
	if err != nil {
		return nil, xerr.Wrapf(err, "创建 Elasticsearch 客户端失败")
	}
	return &Client{raw: raw}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	resp, err := c.raw.Info(c.raw.Info.WithContext(ctx))
	if err != nil {
		return xerr.Wrapf(err, "连接 Elasticsearch 失败")
	}
	defer resp.Body.Close()
	if resp.IsError() {
		return xerr.Internal("连接 Elasticsearch 失败", responseError(resp))
	}
	return nil
}

func (c *Client) Index(ctx context.Context, index, id string, document any) (map[string]any, error) {
	body, err := jsonBody(document)
	if err != nil {
		return nil, err
	}
	resp, err := c.raw.Index(index, body, c.raw.Index.WithContext(ctx), c.raw.Index.WithDocumentID(id))
	if err != nil {
		return nil, xerr.Wrapf(err, "写入 Elasticsearch 文档失败: index=%s id=%s", index, id)
	}
	return decodeResponse(resp, "写入 Elasticsearch 文档失败")
}

func (c *Client) Get(ctx context.Context, index, id string) (map[string]any, error) {
	resp, err := c.raw.Get(index, id, c.raw.Get.WithContext(ctx))
	if err != nil {
		return nil, xerr.Wrapf(err, "读取 Elasticsearch 文档失败: index=%s id=%s", index, id)
	}
	return decodeResponse(resp, "读取 Elasticsearch 文档失败")
}

func (c *Client) Search(ctx context.Context, index string, query any) (map[string]any, error) {
	body, err := jsonBody(query)
	if err != nil {
		return nil, err
	}
	resp, err := c.raw.Search(c.raw.Search.WithContext(ctx), c.raw.Search.WithIndex(index), c.raw.Search.WithBody(body))
	if err != nil {
		return nil, xerr.Wrapf(err, "查询 Elasticsearch 失败: index=%s", index)
	}
	return decodeResponse(resp, "查询 Elasticsearch 失败")
}

func (c *Client) Delete(ctx context.Context, index, id string) (map[string]any, error) {
	resp, err := c.raw.Delete(index, id, c.raw.Delete.WithContext(ctx))
	if err != nil {
		return nil, xerr.Wrapf(err, "删除 Elasticsearch 文档失败: index=%s id=%s", index, id)
	}
	return decodeResponse(resp, "删除 Elasticsearch 文档失败")
}

func (c *Client) Raw() *elasticsearch.Client {
	return c.raw
}

func jsonBody(value any) (io.Reader, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, xerr.Wrapf(err, "编码 Elasticsearch 请求失败")
	}
	return bytes.NewReader(data), nil
}

func decodeResponse(resp *esapi.Response, message string) (map[string]any, error) {
	defer resp.Body.Close()
	if resp.IsError() {
		return nil, xerr.Internal(message, responseError(resp))
	}
	var out map[string]any
	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&out); err != nil {
		return nil, xerr.Wrapf(err, "解析 Elasticsearch 响应失败")
	}
	return out, nil
}

func responseError(resp *esapi.Response) error {
	data, _ := io.ReadAll(resp.Body)
	if len(data) == 0 {
		return io.ErrUnexpectedEOF
	}
	return errors.New(string(data))
}

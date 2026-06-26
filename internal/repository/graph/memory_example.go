package graph

import (
	"context"
	"fmt"

	dbneo4j "github.com/boxify/api-go/internal/infrastructure/db/neo4j"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/xerr"
)

const (
	upsertMemoryExampleCypher = `
MERGE (m:MemoryExample {user_id: $user_id, id: $id})
SET m += $props
RETURN m { .id, .user_id, .text } AS memory`

	findMemoryExampleCypher = `
MATCH (m:MemoryExample {user_id: $user_id, id: $id})
RETURN m { .id, .user_id, .text } AS memory`

	deleteMemoryExampleCypher = `
MATCH (m:MemoryExample {user_id: $user_id, id: $id})
WITH collect(m) AS nodes, count(m) AS deleted
FOREACH (m IN nodes | DETACH DELETE m)
RETURN deleted`
)

type MemoryExampleRepository struct {
	client *dbneo4j.Client
}

func NewMemoryExampleRepository(client *dbneo4j.Client) repository.MemoryExampleRepository {
	if client == nil {
		panic("neo4j client is required")
	}
	return &MemoryExampleRepository{client: client}
}

func (r *MemoryExampleRepository) Upsert(ctx context.Context, item repository.MemoryExample) (repository.MemoryExample, error) {
	params, err := memoryExampleParams(item)
	if err != nil {
		return repository.MemoryExample{}, xerr.Internal("构建 Neo4j 记忆示例参数失败", err)
	}
	rows, err := r.client.Write(ctx, upsertMemoryExampleCypher, params)
	if err != nil {
		return repository.MemoryExample{}, xerr.Wrapf(err, "保存 Neo4j 记忆示例失败")
	}
	if len(rows) == 0 {
		return repository.MemoryExample{}, xerr.Internal("保存 Neo4j 记忆示例失败", fmt.Errorf("empty result"))
	}
	out, err := dbneo4j.DecodeOne[repository.MemoryExample](rows, "memory")
	if err != nil {
		return repository.MemoryExample{}, xerr.Internal("解析 Neo4j 记忆示例失败", err)
	}
	return out, nil
}

func (r *MemoryExampleRepository) FindByID(ctx context.Context, userID string, id string) (repository.MemoryExample, error) {
	rows, err := r.client.Read(ctx, findMemoryExampleCypher, map[string]any{
		"user_id": userID,
		"id":      id,
	})
	if err != nil {
		return repository.MemoryExample{}, xerr.Wrapf(err, "查询 Neo4j 记忆示例失败")
	}
	if len(rows) == 0 {
		return repository.MemoryExample{}, xerr.NotFound("记忆不存在")
	}
	item, err := dbneo4j.DecodeOne[repository.MemoryExample](rows, "memory")
	if err != nil {
		return repository.MemoryExample{}, xerr.Internal("解析 Neo4j 记忆示例失败", err)
	}
	return item, nil
}

func (r *MemoryExampleRepository) Delete(ctx context.Context, userID string, id string) error {
	_, err := r.client.Write(ctx, deleteMemoryExampleCypher, map[string]any{
		"user_id": userID,
		"id":      id,
	})
	if err != nil {
		return xerr.Wrapf(err, "删除 Neo4j 记忆示例失败")
	}
	return nil
}

func memoryExampleParams(item repository.MemoryExample) (map[string]any, error) {
	props, err := dbneo4j.Encode(item)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id":      item.ID,
		"user_id": item.UserID,
		"props":   props,
	}, nil
}

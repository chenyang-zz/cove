/**
 * @Time   : 2026/6/24 23:48
 * @Author : chenyangzhao542@gmail.com
 * @File   : memory_community.go
 **/

package graph

import (
	"context"
	"log/slog"
	"time"

	"github.com/boxify/api-go/internal/core/memory"
	dbneo4j "github.com/boxify/api-go/internal/infrastructure/db/neo4j"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/xerr"
)

type MemoryCommunityRepository struct {
	log    *slog.Logger
	client *dbneo4j.Client
}

func NewMemoryCommunityRepository(client *dbneo4j.Client) repository.MemoryCommunityRepository {
	if client == nil {
		panic("neo4j client is required")
	}
	return &MemoryCommunityRepository{
		log:    xlog.Component("memory_community_repository"),
		client: client,
	}
}

// HasCommunities 查询用户是否存在社区
func (m *MemoryCommunityRepository) HasCommunities(ctx context.Context, userId string) (bool, error) {
	rows, err := m.client.Read(ctx, countCommunitiesCypher, map[string]any{
		"user_id": userId,
	})
	if err != nil {
		return false, xerr.Wrapf(err, "查询用户是否存在社区失败")
	}
	res, err := dbneo4j.DecodeOne[int](rows, "cnt")
	if err != nil {
		return false, err
	}
	return res > 0, nil
}

// ListEntityEmbedding 查询用户实体向量
func (m *MemoryCommunityRepository) ListEntityEmbedding(ctx context.Context, userId string) ([]*memory.EntityEmbedding, error) {
	rows, err := m.client.Read(ctx, listEntityEmbeddingCypher, map[string]any{
		"user_id": userId,
	})
	if err != nil {
		return nil, xerr.Wrapf(err, "查询用户是否存在社区失败")
	}
	return dbneo4j.DecodeMany[*memory.EntityEmbedding](rows, "entity_embedding")
}

// ListNeighborsForVote 根据实体id查询用于投票的邻居节点
func (m *MemoryCommunityRepository) ListNeighborsForVote(ctx context.Context, userId string,
	entityIds []string) (map[string][]*memory.EntityNeighborForVote,
	error) {
	rows, err := m.client.Read(ctx, listNeighborsByEntityIdsCypher, map[string]any{
		"user_id":    userId,
		"entity_ids": entityIds,
	})
	if err != nil {
		return nil, xerr.Wrapf(err, "查询用于投票的邻居节点失败")
	}
	list, err := dbneo4j.DecodeMany[*memory.EntityNeighborForVote](rows, "entity_embedding")
	if err != nil {
		return nil, err
	}

	res := make(map[string][]*memory.EntityNeighborForVote)
	for _, item := range list {
		res[item.ID] = append(res[item.ID], item)
	}

	return res, nil
}

// UpsertCommunities 插入社区节点
func (m *MemoryCommunityRepository) UpsertCommunities(ctx context.Context, userId string, communityIds []string) error {
	_, err := m.client.Write(ctx, upsertCommunitiesCypher, map[string]any{
		"community_ids": communityIds,
		"user_id":       userId,
		"created_at":    time.Now(),
	})
	if err != nil {
		return xerr.Wrapf(err, "插入社区节点失败")
	}
	return nil
}

// AssignEntityToCommunity 将节点关联到指定社区
func (m *MemoryCommunityRepository) AssignEntityToCommunity(ctx context.Context, userId string, entityIds []string, communityIds []string) error {
	rows := make([]map[string]string, 0, len(entityIds))
	for i := 0; i < len(entityIds); i++ {
		rows = append(rows, map[string]string{
			"entity_id":    entityIds[i],
			"community_id": communityIds[i],
		})
	}
	_, err := m.client.Write(ctx, assignEntitiesToCommunitiesCypher, map[string]any{
		"user_id": userId,
		"rows":    rows,
	})
	if err != nil {
		return xerr.Wrapf(err, "节点关联到指定社区失败")
	}
	return nil
}

// RefreshCommunityMemberCount 更新社区成员数量
func (m *MemoryCommunityRepository) RefreshCommunityMemberCount(ctx context.Context, userId string, communityIds []string) ([]int, error) {
	rows, err := m.client.Write(ctx, refreshCommunityMemberCountCypher, map[string]any{
		"user_id":       userId,
		"community_ids": communityIds,
	})
	if err != nil {
		return nil, xerr.Wrapf(err, "更新社区成员数量失败")
	}

	return dbneo4j.DecodeMany[int](rows, "cnt")
}

// GetCommunityMembers 获取社区成员
func (m *MemoryCommunityRepository) GetCommunityMembers(ctx context.Context, userId string, communityIds []string) ([][]*memory.CommunityMember, error) {
	rows, err := m.client.Read(ctx, getCommunityMembersCypher, map[string]any{
		"user_id":       userId,
		"community_ids": communityIds,
	})
	if err != nil {
		return nil, xerr.Wrapf(err, "获取社区成员失败")
	}
	return dbneo4j.DecodeMany[[]*memory.CommunityMember](rows, "members")
}

// PruneEmptyCommunity 清理没有成员的社区节点
func (m *MemoryCommunityRepository) PruneEmptyCommunity(ctx context.Context, userId string) error {
	_, err := m.client.Write(ctx, pruneEmptyCommunityCypher, map[string]any{
		"user_id": userId,
	})
	if err != nil {
		return xerr.Wrap(err, "清理没有成员的社区节点失败")
	}
	return nil
}

// UpdateCommunityMetadata 更新社区元数据
func (m *MemoryCommunityRepository) UpdateCommunityMetadata(ctx context.Context, userId string, updateItems []*memory.UpdateCommunityMetaItem) error {
	rows, err := dbneo4j.EncodeRows(updateItems)
	if err != nil {
		return err
	}
	_, err = m.client.Write(ctx, updateCommunityMetadataCypher, map[string]any{
		"user_id": userId,
		"rows":    rows,
	})

	if err != nil {
		return xerr.Wrap(err, "更新社区元数据失败")
	}

	return nil
}

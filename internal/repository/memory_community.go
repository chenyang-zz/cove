/**
 * @Time   : 2026/6/24 23:47
 * @Author : chenyangzhao542@gmail.com
 * @File   : memory_community.go
 **/

package repository

import (
	"context"

	"github.com/boxify/api-go/internal/core/memory"
)

type MemoryCommunityRepository interface {
	HasCommunities(ctx context.Context, userId string) (bool, error)
	ListEntityEmbedding(ctx context.Context, userId string) ([]*memory.EntityEmbedding, error)
	ListNeighborsForVote(ctx context.Context, userId string, entityIds []string) (map[string][]*memory.EntityNeighborForVote, error)
	UpsertCommunities(ctx context.Context, userId string, communityIds []string) error
	AssignEntityToCommunity(ctx context.Context, userId string, entityIds []string, communityIds []string) error
	RefreshCommunityMemberCount(ctx context.Context, userId string, communityIds []string) ([]int, error)
	GetCommunityMembers(ctx context.Context, userId string, communityIds []string) ([][]*memory.CommunityMember, error)
	PruneEmptyCommunity(ctx context.Context, userId string) error
	UpdateCommunityMetadata(ctx context.Context, userId string, updateItems []*memory.UpdateCommunityMetaItem) error
}

/**
 * @Time   : 2026/6/23 22:58
 * @Author : chenyangzhao542@gmail.com
 * @File   : memory_graph.go
 **/

package repository

import (
	"context"

	"github.com/boxify/api-go/internal/core/memory"
)

type MemoryGraphRepository interface {
	ListEntitiesByType(ctx context.Context, userId string, entityType string) ([]*memory.EntityNode, error)
	SaveGraph(
		ctx context.Context,
		dialogues []*memory.DialogueNode,
		chunks []*memory.ChunkNode,
		statements []*memory.StatementNode,
		entities []*memory.EntityNode,
		events []*memory.EventNode,
		mentions []*memory.MentionEdge,
		relations []*memory.RelationEdge,
		involves []*memory.InvolvesEdge,
	) error
}

/**
 * @Time   : 2026/6/23 22:58
 * @Author : chenyangzhao542@gmail.com
 * @File   : memory_graph.go
 **/

package graph

import (
	"context"
	"log/slog"

	"github.com/boxify/api-go/internal/core/memory"
	dbneo4j "github.com/boxify/api-go/internal/infrastructure/db/neo4j"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/xerr"
)

type MemoryGraphRepository struct {
	log    *slog.Logger
	client *dbneo4j.Client
}

func NewMemoryGraphRepository(client *dbneo4j.Client) repository.MemoryGraphRepository {
	if client == nil {
		panic("neo4j client is required")
	}
	return &MemoryGraphRepository{
		log:    xlog.Component("memory_graph_repository"),
		client: client,
	}
}

// ListEntitiesByType 取用户已有同类实体
func (m *MemoryGraphRepository) ListEntitiesByType(ctx context.Context, userId string, entityType string) ([]*memory.EntityNode, error) {
	rows, err := m.client.Read(ctx, listEntitiesByTypeCypher, map[string]any{
		"user_id": userId,
		"type":    entityType,
	})

	if err != nil {
		return nil, xerr.Wrapf(err, "查找同类型实体失败")
	}

	return dbneo4j.DecodeMany[*memory.EntityNode](rows, "entity")
}

// SaveGraph 保存图谱
func (m *MemoryGraphRepository) SaveGraph(ctx context.Context, dialogues []*memory.DialogueNode, chunks []*memory.ChunkNode, statements []*memory.StatementNode, entities []*memory.EntityNode, events []*memory.EventNode, mentions []*memory.MentionEdge, relations []*memory.RelationEdge, involves []*memory.InvolvesEdge) error {
	err := m.client.WriteTx(ctx, func(tx dbneo4j.Tx) error {
		// 保存来源节点
		if dialogues != nil {
			rows, err := dbneo4j.Encode(dialogues)
			if err != nil {
				return err
			}
			_, err = tx.Run(ctx, saveDialoguesCypher, map[string]any{
				"rows": rows,
			})
			if err != nil {
				return xerr.Wrapf(err, "保存来源根节点失败")
			}
		}

		// 保存切片节点
		if chunks != nil {
			rows, err := dbneo4j.Encode(chunks)
			if err != nil {
				return err
			}
			_, err = tx.Run(ctx, saveChunksCypher, map[string]any{
				"rows": rows,
			})
			if err != nil {
				return xerr.Wrapf(err, "保存来源切片失败")
			}
		}

		// 保存陈述节点
		if statements != nil {
			rows, err := dbneo4j.Encode(statements)
			if err != nil {
				return err
			}
			_, err = tx.Run(ctx, saveStatementsCypher, map[string]any{
				"rows": rows,
			})
			if err != nil {
				return xerr.Wrapf(err, "保存陈述失败")
			}
		}

		// 保存实体节点
		if entities != nil {
			rows, err := dbneo4j.Encode(entities)
			if err != nil {
				return err
			}
			_, err = tx.Run(ctx, saveEntitiesCypher, map[string]any{
				"rows": rows,
			})
			if err != nil {
				return xerr.Wrapf(err, "保存实体失败")
			}
		}

		// 保存事件节点
		if events != nil {
			rows, err := dbneo4j.Encode(events)
			if err != nil {
				return err
			}
			_, err = tx.Run(ctx, saveEventsCypher, map[string]any{
				"rows": rows,
			})
			if err != nil {
				return xerr.Wrapf(err, "保存事件失败")
			}
		}

		// 保存引用边
		if mentions != nil {
			rows, err := dbneo4j.Encode(mentions)
			if err != nil {
				return err
			}
			_, err = tx.Run(ctx, saveMentionsCypher, map[string]any{
				"rows": rows,
			})
			if err != nil {
				return xerr.Wrapf(err, "保存引用边失败")
			}
		}

		// 保存关系边
		if relations != nil {
			rows, err := dbneo4j.Encode(relations)
			if err != nil {
				return err
			}
			_, err = tx.Run(ctx, saveRelationsCypher, map[string]any{
				"rows": rows,
			})
			if err != nil {
				return xerr.Wrapf(err, "保存关系边失败")
			}
		}

		// 保存事件参与边
		if involves != nil {
			rows, err := dbneo4j.Encode(involves)
			if err != nil {
				return err
			}
			_, err = tx.Run(ctx, saveInvolvesCypher, map[string]any{
				"rows": rows,
			})
			if err != nil {
				return xerr.Wrapf(err, "保存事件参与边失败")
			}
		}

		return nil
	})

	if err != nil {
		return xerr.Wrapf(err, "保存图谱失败")
	}

	m.log.InfoContext(ctx, "记忆图谱写入成功",
		slog.Int("dialogue", len(dialogues)),
		slog.Int("chunk", len(chunks)),
		slog.Int("statement", len(statements)),
		slog.Int("entity", len(entities)),
		slog.Int("mention", len(mentions)),
		slog.Int("relation", len(relations)),
		slog.Int("involve", len(involves)),
	)

	return nil
}

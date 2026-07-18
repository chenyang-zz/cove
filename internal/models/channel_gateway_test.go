package models

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

// TestChannelGatewayTenantIndexes 验证账号和 Inbox 提供组合外键所需的唯一租户索引。
func TestChannelGatewayTenantIndexes(t *testing.T) {
	tests := []struct {
		name       string
		model      any
		indexName  string
		wantFields []string
	}{
		{
			name:       "account tenant",
			model:      &ChannelAccount{},
			indexName:  "uq_channel_accounts_id_user_id",
			wantFields: []string{"ID", "UserID"},
		},
		{
			name:       "inbox scope",
			model:      &ChannelInboxEvent{},
			indexName:  "uq_channel_inbox_events_scope",
			wantFields: []string{"ID", "AccountID", "UserID"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := schema.Parse(tt.model, &sync.Map{}, schema.NamingStrategy{})
			if err != nil {
				t.Fatalf("schema.Parse(%T) error = %v, want nil", tt.model, err)
			}
			var found *schema.Index
			for _, index := range parsed.ParseIndexes() {
				if index.Name == tt.indexName {
					found = index
					break
				}
			}
			if found == nil {
				t.Fatalf("schema.Parse(%T) missing index %q", tt.model, tt.indexName)
			}
			if found.Class != "UNIQUE" {
				t.Fatalf("index %s class = %q, want UNIQUE", tt.indexName, found.Class)
			}
			if len(found.Fields) != len(tt.wantFields) {
				t.Fatalf("index %s fields = %d, want %d", tt.indexName, len(found.Fields), len(tt.wantFields))
			}
			for i, want := range tt.wantFields {
				if got := found.Fields[i].Field.Name; got != want {
					t.Fatalf("index %s field[%d] = %q, want %q", tt.indexName, i, got, want)
				}
			}
		})
	}
}

// TestChannelGatewayTenantRelationships 验证带 UserID 的网关子模型使用组合外键阻止跨用户关联。
func TestChannelGatewayTenantRelationships(t *testing.T) {
	tests := []struct {
		name          string
		model         any
		relationName  string
		constraint    string
		wantForeign   []string
		wantReference []string
	}{
		{
			name:          "binding account",
			model:         &ChannelBinding{},
			relationName:  "Account",
			constraint:    "fk_channel_bindings_account_tenant",
			wantForeign:   []string{"AccountID", "UserID"},
			wantReference: []string{"ID", "UserID"},
		},
		{
			name:          "inbox account",
			model:         &ChannelInboxEvent{},
			relationName:  "Account",
			constraint:    "fk_channel_inbox_events_account_tenant",
			wantForeign:   []string{"AccountID", "UserID"},
			wantReference: []string{"ID", "UserID"},
		},
		{
			name:          "outbox account",
			model:         &ChannelOutboxMessage{},
			relationName:  "Account",
			constraint:    "fk_channel_outbox_messages_account_tenant",
			wantForeign:   []string{"AccountID", "UserID"},
			wantReference: []string{"ID", "UserID"},
		},
		{
			name:          "outbox inbox",
			model:         &ChannelOutboxMessage{},
			relationName:  "InboxEvent",
			constraint:    "fk_channel_outbox_messages_inbox_scope",
			wantForeign:   []string{"InboxEventID", "AccountID", "UserID"},
			wantReference: []string{"ID", "AccountID", "UserID"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := schema.Parse(tt.model, &sync.Map{}, schema.NamingStrategy{})
			if err != nil {
				t.Fatalf("schema.Parse(%T) error = %v, want nil", tt.model, err)
			}
			relation := parsed.Relationships.Relations[tt.relationName]
			if relation == nil {
				t.Fatalf("schema.Parse(%T) missing relation %q", tt.model, tt.relationName)
			}
			if len(relation.References) != len(tt.wantForeign) {
				t.Fatalf("relation %s references = %d, want %d", tt.relationName, len(relation.References), len(tt.wantForeign))
			}
			for i, reference := range relation.References {
				if got := reference.ForeignKey.Name; got != tt.wantForeign[i] {
					t.Fatalf("relation %s foreign[%d] = %q, want %q", tt.relationName, i, got, tt.wantForeign[i])
				}
				if got := reference.PrimaryKey.Name; got != tt.wantReference[i] {
					t.Fatalf("relation %s reference[%d] = %q, want %q", tt.relationName, i, got, tt.wantReference[i])
				}
			}
			foreignKey := relation.ParseConstraint()
			if foreignKey == nil {
				t.Fatalf("relation %s constraint = nil", tt.relationName)
			}
			if foreignKey.Name != tt.constraint {
				t.Fatalf("relation %s constraint = %q, want %q", tt.relationName, foreignKey.Name, tt.constraint)
			}
			if foreignKey.OnDelete != "CASCADE" || foreignKey.OnUpdate != "RESTRICT" {
				t.Fatalf("relation %s actions = update %q delete %q, want RESTRICT/CASCADE", tt.relationName, foreignKey.OnUpdate, foreignKey.OnDelete)
			}
		})
	}
}

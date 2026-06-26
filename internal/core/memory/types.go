/**
 * @Time   : 2026/6/23 00:30
 * @Author : chenyangzhao542@gmail.com
 * @File   : types.go
 **/

package memory

import "time"

type ExtractionStats struct {
	DialogueID     string   `json:"dialogue_id"`
	ChunkCount     int      `json:"chunk_count"`
	StatementCount int      `json:"statement_count"`
	EntityCount    int      `json:"entity_count"`
	RelationCount  int      `json:"relation_count"`
	EventCount     int      `json:"event_count"`
	EntityIDs      []string `json:"entity_ids"`
}

type DialogueSourceType string

const (
	ManualDialogueSource DialogueSourceType = "manual"
	AutoDialogueSource   DialogueSourceType = "auto"
)

// DialogueNode 一次萃取的来源根节点：一段对话或一条主动记住的文本
type DialogueNode struct {
	ID              string             `json:"id"`
	UserID          string             `json:"user_id"`
	Content         string             `json:"content"` // 来源全文
	Source          DialogueSourceType `json:"source"`
	SourceMessageID string             `json:"source_message_id"`
	DialogAt        time.Time          `json:"dialog_at"`
	CreatedAt       time.Time          `json:"created_at"`
}

type SpeakerType string

const (
	UserSpeaker      SpeakerType = "user"
	AssistantSpeaker SpeakerType = "assistant"
)

// ChunkNode 对话片段：按轮次 / token 切片，承上启下用于朔源
type ChunkNode struct {
	ID        string      `json:"id"`
	UserID    string      `json:"user_id"`
	DialogID  string      `json:"dialog_id"`
	Content   string      `json:"content"`
	Speaker   SpeakerType `json:"speaker"` // user | assistant
	Sequence  int         `json:"sequence"`
	CreatedAt time.Time   `json:"created_at"`
}

type StmtType string

const (
	StmtFact       StmtType = "FACT"
	StmtOpinion    StmtType = "OPINION"
	StmtPrediction StmtType = "PREDICTION"
	StmtSuggestion StmtType = "SUGGESTION"
)

type TemporalType string

const (
	TemporalStatic    TemporalType = "STATIC"
	TemporalDynamic   TemporalType = "DYNAMIC"
	TemporalAtemporal TemporalType = "ATEMPORAL"
)

type LayerType string

const (
	LayerShortTerm LayerType = "LAYER_SHORT_TERM"
	LayerLongTerm  LayerType = "LAYER_LONG_TERM"
)

// StatementNode 原子陈述句：萃取的最小语义单元，带类型与时间属性
type StatementNode struct {
	ID           string       `json:"id"`
	UserID       string       `json:"user_id"`
	ChunkID      string       `json:"chunk_id"`
	Statement    string       `json:"statement"`
	StmtType     StmtType     `json:"stmt_type"`
	TemporalType TemporalType `json:"temporal_type"`
	Speaker      SpeakerType  `json:"speaker"`
	ValidAt      time.Time    `json:"valid_at"`
	InvalidAt    time.Time    `json:"invalid_at"`
	DialogAt     time.Time    `json:"dialog_at"`
	Embedding    []float64    `json:"embedding"`
	// 记忆动力学
	Importance   float64   `json:"importance"` // default: 0.5
	Confidence   float64   `json:"confidence"` // default: 0.8
	MemoryLayer  LayerType `json:"memory_layer"`
	AccessCount  int       `json:"access_count"`
	LastAccessAt time.Time `json:"last_access_at"`
	// 情绪（含情绪时填，与 PG 情绪表并存）
	HasEmotionalState bool      `json:"has_emotional_state"`
	EmotionType       string    `json:"emotion_type"`
	EmotionIntensity  float64   `json:"emotion_intensity"`
	EmotionKeywords   []string  `json:"emotion_keywords"`
	CreatedAt         time.Time `json:"created_at"`
}

type ConnectStrengthType string

const (
	StrongConnectStrength ConnectStrengthType = "string"
	WeakConnectStrength   ConnectStrengthType = "weak"
	BothConnectStrength   ConnectStrengthType = "both"
)

// EntityNode 实体节点：萃取出的画像实体（人/组织/知识/偏好/目标等）
type EntityNode struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"` // 受控实体类型中文标签
	Description   string    `json:"description"`
	Aliases       []string  `json:"aliases"`
	NameEmbedding []float64 `json:"name_embedding"`
	CommunityID   string    `json:"community_id"`
	// 记忆动力学
	Importance      float64             `json:"importance"` // default: 0.5
	Confidence      float64             `json:"confidence"` // default: 0.8
	MemoryLayer     LayerType           `json:"memory_layer"`
	AccessCount     int                 `json:"access_count"`
	LastAccessAt    time.Time           `json:"last_access_at"`
	MentionCount    int                 `json:"mention_count"` // default: 1
	ConnectStrength ConnectStrengthType `json:"connect_strength"`
	// 画像增强（巩固任务 LLM 会写）
	CoreFacts          []string  `json:"core_facts"`
	Traits             []string  `json:"traits"`
	LastConsolidatedAt time.Time `json:"last_consolidated_at"`
	CreatedAt          time.Time `json:"created_at"`
}

// EventNode 事件节点: 带 event_time, 供记忆时间线展示
type EventNode struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	EventTime   time.Time `json:"event_time"`
	Embedding   []float64 `json:"embedding"`
	CreatedAt   time.Time `json:"created_at"`
}

// RelationEdge 实体间三元组关系： Entity -> [:RELATION] -> Entity
type RelationEdge struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	SourceID         string    `json:"source_id"`         // 主语实体
	TargetID         string    `json:"target_id"`         // 宾语实体
	Predicate        string    `json:"predicate"`         // 受控谓词中文标签
	PredicateSurface string    `json:"predicate_surface"` // 原文中的关系表达
	SourceText       string    `json:"source_text"`       // 关系来源陈述句原文
	StatementID      string    `json:"statement_id"`
	Value            string    `json:"value"` // 附加值（数量、内容）
	ValidAt          time.Time `json:"valid_at"`
	InvalidAt        time.Time `json:"invalid_at"`
	Importance       float64   `json:"importance"` // default: 0.5
	Confidence       float64   `json:"confidence"` // default: 0.8
	AccessCount      int       `json:"access_count"`
	CreatedAt        time.Time `json:"created_at"`
}

// MentionEdge 陈述提及实体：Statement -> [:MENTIONS] -> Entity
type MentionEdge struct {
	UserID          string              `json:"user_id"`
	StatementID     string              `json:"statement_id"`
	EntityID        string              `json:"entity_id"`
	ConnectStrength ConnectStrengthType `json:"connect_strength"` // default: strong
	CreatedAt       time.Time           `json:"created_at"`
}

// InvolvesEdge 事件涉及实体： Event -> [:INVOLVES] -> Entity
type InvolvesEdge struct {
	UserID    string    `json:"user_id"`
	EventID   string    `json:"event_id"`
	EntityID  string    `json:"entity_id"`
	Role      string    `json:"role"` // 暂未使用
	CreatedAt time.Time `json:"created_at"`
}

type ExtractedStatement struct {
	Statement            string       `json:"statement"`
	StatementType        StmtType     `json:"statement_type"`
	TemporalType         TemporalType `json:"temporal_type"`
	HasUnsolvedReference bool         `json:"has_unsolved_reference"`
	// 记忆动力学
	Importance float64 `json:"importance"` // default: 0.5
	Confidence float64 `json:"confidence"` // default: 0.8
	// 情绪
	HasEmotionalState bool     `json:"has_emotional_state"`
	EmotionType       string   `json:"emotion_type"`
	EmotionIntensity  float64  `json:"emotion_intensity"`
	EmotionKeywords   []string `json:"emotion_keywords"`
}

type StatementExtractionResult struct {
	Statements []*ExtractedStatement `json:"statements"`
}

type TripletExtractionResult struct {
	Entities []*ExtractedEntity  `json:"entities"`
	Triplets []*ExtractedTriplet `json:"triplets"`
	Events   []*ExtractedEvent   `json:"events"`
}

// ExtractedEvent LLM 抽取的事件（一次性发生、有时间的经历）
type ExtractedEvent struct {
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	EventTime    time.Time `json:"event_time"`
	Participants []string  `json:"participants"`
}

// ExtractedEntity LLM 抽取的实体（chunk 内局部 idx 用于关联 triplet）
type ExtractedEntity struct {
	EntityIdx   int     `json:"entity_idx"` // default: -1
	Name        string  `json:"name"`
	Type        string  `json:"type"` // default: 其他
	Description string  `json:"description"`
	Importance  float64 `json:"importance"` // default: 0.5
	Confidence  float64 `json:"confidence"` // default: 0.8
}

// ExtractedTriplet LLM输出的三元组
type ExtractedTriplet struct {
	SubjectName      string    `json:"subject_name"`
	SubjectID        int       `json:"subject_id"` // default: -1
	Predicate        string    `json:"predicate"`  // default：关联于
	PredicateSurface string    `json:"predicate_surface"`
	ObjectName       string    `json:"object_name"`
	ObjectID         int       `json:"object_id"` // default: -1
	Value            string    `json:"value"`
	ValidAt          time.Time `json:"valid_at"`
	InvalidAt        time.Time `json:"invalid_at"`
	Importance       float64   `json:"importance"` // default: 0.5
	Confidence       float64   `json:"confidence"` // default: 0.8
}

// DedupDecision 实体去重判定
type DedupDecision struct {
	SameEntity   bool    `json:"same_entity"`
	CanonicalIdx int     `json:"canonical_idx"`
	Confidence   float64 `json:"confidence"`
	Reason       string  `json:"reason"`
}

type EntityNeighborForVote struct {
	ID            string    `json:"id"`
	EntityID      string    `json:"entity_id"`
	CommunityID   string    `json:"community_id"`
	NameEmbedding []float64 `json:"name_embedding"`
}

type EntityEmbedding struct {
	ID            string    `json:"id"`
	NameEmbedding []float64 `json:"name_embedding"`
	CommunityID   string    `json:"community_id"`
}

type CommunityMember struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	Description   string    `json:"description"`
	Aliases       []string  `json:"aliases"`
	NameEmbedding []float64 `json:"name_embedding"`
}

type CommunityMetadata struct {
	Name    string `json:"name"`
	Summary string `json:"summary"`
}

type UpdateCommunityMetaItem struct {
	CommunityId string `json:"community_id"`
	Name        string `json:"name"`
	Summary     string `json:"summary"`
}

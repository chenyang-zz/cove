/**
 * @Time   : 2026/6/23 01:32
 * @Author : chenyangzhao542@gmail.com
 * @File   : memory.go
 **/

package prompt

import "fmt"

type StatementExtractData struct {
	Content string
	Context string
}

type TripletExtractData struct {
	Statemnet   string
	Context     string
	EntityTypes []string
	Predicates  []string
	ValidAt     string
	InvalidAt   string
	DialogAt    string
}

type DedupEntity struct {
	Name        string
	Type        string
	Description string
	Aliases     []string
}

type DedupContext struct {
	NameTextSim  float64
	NameEmbedSim float64
	NameContains bool
}

type DedupEntityData struct {
	EntityA DedupEntity
	EntityB DedupEntity
	Context DedupContext
}

type CommunityMember struct {
	Name        string
	Description string
}

type GenerateCommunityMetadataData struct {
	Members []*CommunityMember
}

type MemoryPrompts struct {
	namespace string
	manger    *Manager
}

func NewMemoryPrompts(manager *Manager) *MemoryPrompts {
	return &MemoryPrompts{
		namespace: "memory",
		manger:    manager,
	}
}

func (m *MemoryPrompts) StatementExtract(data *StatementExtractData) (string, error) {
	return m._render("statement_extract", data)
}

func (m *MemoryPrompts) TripletExtract(data *TripletExtractData) (string, error) {
	return m._render("triplet_extract", data)
}

func (m *MemoryPrompts) DedupEntity(data *DedupEntityData) (string, error) {
	return m._render("dedup_entity", data)
}

func (m *MemoryPrompts) GenerateCommunityMetadata(data *GenerateCommunityMetadataData) (string, error) {
	return m._render("generate_community_metadata", data)
}

func (m *MemoryPrompts) _render(promptName string, data any) (string, error) {
	return m.manger.Render(fmt.Sprintf("%s/%s", m.namespace, promptName), data)
}

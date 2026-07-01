package domain

import (
	"fmt"

	"github.com/google/uuid"
)

type TaskName string

type QueueName string

const (
	TaskParseDocument     TaskName = "parse:document"
	TaskParseImage        TaskName = "parse:image"
	TaskMemoryExtract     TaskName = "memory:extract"
	TaskMemoryConsolidate TaskName = "memory:consolidate"
	TaskResearchRun       TaskName = "research:run"
)

const (
	QueueDefault  QueueName = "default"
	QueueParse    QueueName = "parse"
	QueueMemory   QueueName = "memory"
	QueueResearch QueueName = "research"
	QueueBeat     QueueName = "beat"
)

type Task struct {
	Name    TaskName
	Queue   QueueName
	Payload any
}

type ParseDocumentPayload struct {
	UserID     uuid.UUID `json:"user_id"`
	DocumentID uuid.UUID `json:"document_id"`
}

func TaskNames() []TaskName {
	return []TaskName{
		TaskParseDocument,
		TaskParseImage,
		TaskMemoryExtract,
		TaskMemoryConsolidate,
		TaskResearchRun,
	}
}

func NewParseDocumentTask(userID uuid.UUID, documentID uuid.UUID) (*Task, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("user_id is required")
	}
	if documentID == uuid.Nil {
		return nil, fmt.Errorf("document_id is required")
	}
	return &Task{
		Name:  TaskParseDocument,
		Queue: QueueParse,
		Payload: &ParseDocumentPayload{
			UserID:     userID,
			DocumentID: documentID,
		},
	}, nil
}

package redis

import (
	"encoding/json"
	"fmt"

	"github.com/boxify/api-go/internal/domain"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

func EncodeTask(task *domain.Task) (*asynq.Task, error) {
	if task == nil {
		return nil, fmt.Errorf("task is nil")
	}
	switch task.Name {
	case domain.TaskParseDocument:
		payload, err := parseDocumentPayload(task.Payload)
		if err != nil {
			return nil, err
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		return asynq.NewTask(string(task.Name), data), nil
	case domain.TaskParseImage, domain.TaskMemoryExtract, domain.TaskMemoryConsolidate, domain.TaskResearchRun:
		return asynq.NewTask(string(task.Name), nil), nil
	default:
		return nil, fmt.Errorf("unknown task name: %s", task.Name)
	}
}

func DecodeTask(task *asynq.Task) (*domain.Task, error) {
	if task == nil {
		return nil, fmt.Errorf("task is nil")
	}
	name := domain.TaskName(task.Type())
	switch name {
	case domain.TaskParseDocument:
		var payload domain.ParseDocumentPayload
		if err := json.Unmarshal(task.Payload(), &payload); err != nil {
			return nil, err
		}
		if payload.UserID == uuid.Nil || payload.DocumentID == uuid.Nil {
			return nil, fmt.Errorf("parse document payload ids are required")
		}
		return &domain.Task{
			Name:    name,
			Queue:   domain.QueueParse,
			Payload: &payload,
		}, nil
	case domain.TaskParseImage:
		return &domain.Task{Name: name, Queue: domain.QueueParse}, nil
	case domain.TaskMemoryExtract:
		return &domain.Task{Name: name, Queue: domain.QueueMemory}, nil
	case domain.TaskMemoryConsolidate:
		return &domain.Task{Name: name, Queue: domain.QueueBeat}, nil
	case domain.TaskResearchRun:
		return &domain.Task{Name: name, Queue: domain.QueueResearch}, nil
	default:
		return nil, fmt.Errorf("unknown task name: %s", name)
	}
}

func parseDocumentPayload(payload any) (*domain.ParseDocumentPayload, error) {
	switch value := payload.(type) {
	case *domain.ParseDocumentPayload:
		if value == nil {
			return nil, fmt.Errorf("parse document payload is nil")
		}
		if value.UserID == uuid.Nil || value.DocumentID == uuid.Nil {
			return nil, fmt.Errorf("parse document payload ids are required")
		}
		return value, nil
	case domain.ParseDocumentPayload:
		if value.UserID == uuid.Nil || value.DocumentID == uuid.Nil {
			return nil, fmt.Errorf("parse document payload ids are required")
		}
		return &value, nil
	default:
		return nil, fmt.Errorf("parse document payload type = %T", payload)
	}
}

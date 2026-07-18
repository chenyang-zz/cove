package redis

import (
	"encoding/json"
	"fmt"

	"github.com/boxify/api-go/internal/domain/types"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

func EncodeTask(task *types.Task) (*asynq.Task, error) {
	if task == nil {
		return nil, fmt.Errorf("task is nil")
	}
	switch task.Name {
	case types.TaskParseDocument:
		payload, err := parseDocumentPayload(task.Payload)
		if err != nil {
			return nil, err
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		return asynq.NewTask(string(task.Name), data), nil
	case types.TaskParseImage:
		payload, err := parseImagePayload(task.Payload)
		if err != nil {
			return nil, err
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		return asynq.NewTask(string(task.Name), data), nil
	case types.TaskGatewayTurn:
		return encodeJSONTask(task.Name, task.Payload)
	case types.TaskGatewayDeliver:
		return encodeJSONTask(task.Name, task.Payload)
	case types.TaskMemoryExtract, types.TaskMemoryConsolidate, types.TaskResearchRun:
		return asynq.NewTask(string(task.Name), nil), nil
	default:
		return nil, fmt.Errorf("unknown task name: %s", task.Name)
	}
}

func DecodeTask(task *asynq.Task) (*types.Task, error) {
	if task == nil {
		return nil, fmt.Errorf("task is nil")
	}
	name := types.TaskName(task.Type())
	switch name {
	case types.TaskParseDocument:
		var payload types.ParseDocumentPayload
		if err := json.Unmarshal(task.Payload(), &payload); err != nil {
			return nil, err
		}
		if payload.UserID == uuid.Nil || payload.DocumentID == uuid.Nil {
			return nil, fmt.Errorf("parse document payload ids are required")
		}
		return &types.Task{
			Name:    name,
			Queue:   types.QueueParse,
			Payload: &payload,
		}, nil
	case types.TaskParseImage:
		var payload types.ParseImagePayload
		if err := json.Unmarshal(task.Payload(), &payload); err != nil {
			return nil, err
		}
		if payload.UserID == uuid.Nil || payload.ImageID == uuid.Nil {
			return nil, fmt.Errorf("parse image payload ids are required")
		}
		return &types.Task{
			Name:    name,
			Queue:   types.QueueParse,
			Payload: &payload,
		}, nil
	case types.TaskGatewayTurn:
		var payload types.GatewayTurnPayload
		if err := json.Unmarshal(task.Payload(), &payload); err != nil {
			return nil, err
		}
		if payload.InboxEventID == uuid.Nil {
			return nil, fmt.Errorf("gateway turn inbox_event_id is required")
		}
		return &types.Task{Name: name, Queue: types.QueueGateway, Payload: &payload}, nil
	case types.TaskGatewayDeliver:
		var payload types.GatewayDeliverPayload
		if err := json.Unmarshal(task.Payload(), &payload); err != nil {
			return nil, err
		}
		if payload.OutboxMessageID == uuid.Nil {
			return nil, fmt.Errorf("gateway deliver outbox_message_id is required")
		}
		return &types.Task{Name: name, Queue: types.QueueGateway, Payload: &payload}, nil
	case types.TaskMemoryExtract:
		return &types.Task{Name: name, Queue: types.QueueMemory}, nil
	case types.TaskMemoryConsolidate:
		return &types.Task{Name: name, Queue: types.QueueBeat}, nil
	case types.TaskResearchRun:
		return &types.Task{Name: name, Queue: types.QueueResearch}, nil
	default:
		return nil, fmt.Errorf("unknown task name: %s", name)
	}
}

func encodeJSONTask(name types.TaskName, payload any) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(string(name), data), nil
}

func parseDocumentPayload(payload any) (*types.ParseDocumentPayload, error) {
	switch value := payload.(type) {
	case *types.ParseDocumentPayload:
		if value == nil {
			return nil, fmt.Errorf("parse document payload is nil")
		}
		if value.UserID == uuid.Nil || value.DocumentID == uuid.Nil {
			return nil, fmt.Errorf("parse document payload ids are required")
		}
		return value, nil
	case types.ParseDocumentPayload:
		if value.UserID == uuid.Nil || value.DocumentID == uuid.Nil {
			return nil, fmt.Errorf("parse document payload ids are required")
		}
		return &value, nil
	default:
		return nil, fmt.Errorf("parse document payload type = %T", payload)
	}
}

func parseImagePayload(payload any) (*types.ParseImagePayload, error) {
	switch value := payload.(type) {
	case *types.ParseImagePayload:
		if value == nil {
			return nil, fmt.Errorf("parse image payload is nil")
		}
		if value.UserID == uuid.Nil || value.ImageID == uuid.Nil {
			return nil, fmt.Errorf("parse image payload ids are required")
		}
		return value, nil
	case types.ParseImagePayload:
		if value.UserID == uuid.Nil || value.ImageID == uuid.Nil {
			return nil, fmt.Errorf("parse image payload ids are required")
		}
		return &value, nil
	default:
		return nil, fmt.Errorf("parse image payload type = %T", payload)
	}
}

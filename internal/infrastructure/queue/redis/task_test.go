package redis_test

import (
	"context"
	"testing"

	"github.com/boxify/api-go/internal/domain/types"
	"github.com/boxify/api-go/internal/infrastructure/queue"
	queueredis "github.com/boxify/api-go/internal/infrastructure/queue/redis"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

func TestEncodeDecodeParseDocumentTaskRoundTrip(t *testing.T) {
	// 验证 Redis/Asynq 实现能复用 domain 任务契约完成编码和解码。
	userID := uuid.New()
	documentID := uuid.New()
	task, err := types.NewParseDocumentTask(userID, documentID)
	if err != nil {
		t.Fatalf("NewParseDocumentTask error = %v", err)
	}

	asynqTask, err := queueredis.EncodeTask(task)
	if err != nil {
		t.Fatalf("EncodeTask error = %v", err)
	}
	if asynqTask.Type() != string(types.TaskParseDocument) {
		t.Fatalf("task type = %q, want %q", asynqTask.Type(), types.TaskParseDocument)
	}

	got, err := queueredis.DecodeTask(asynqTask)
	if err != nil {
		t.Fatalf("DecodeTask error = %v", err)
	}
	payload, ok := got.Payload.(*types.ParseDocumentPayload)
	if !ok {
		t.Fatalf("payload type = %T, want *types.ParseDocumentPayload", got.Payload)
	}
	if payload.UserID != userID || payload.DocumentID != documentID {
		t.Fatalf("payload = %+v, want user/document ids", payload)
	}
}

// 验证图片解析任务能完成 Redis/Asynq 编码和解码往返。
func TestEncodeDecodeParseImageTaskRoundTrip(t *testing.T) {
	userID := uuid.New()
	imageID := uuid.New()
	task, err := types.NewParseImageTask(userID, imageID)
	if err != nil {
		t.Fatalf("NewParseImageTask error = %v", err)
	}

	asynqTask, err := queueredis.EncodeTask(task)
	if err != nil {
		t.Fatalf("EncodeTask error = %v", err)
	}
	if asynqTask.Type() != string(types.TaskParseImage) {
		t.Fatalf("task type = %q, want %q", asynqTask.Type(), types.TaskParseImage)
	}

	got, err := queueredis.DecodeTask(asynqTask)
	if err != nil {
		t.Fatalf("DecodeTask error = %v", err)
	}
	payload, ok := got.Payload.(*types.ParseImagePayload)
	if !ok {
		t.Fatalf("payload type = %T, want *types.ParseImagePayload", got.Payload)
	}
	if payload.UserID != userID || payload.ImageID != imageID {
		t.Fatalf("payload = %+v, want user/image ids", payload)
	}
}

// TestEncodeDecodeGatewayTasksRoundTrip 验证网关回合和投递任务保留稳定 ID 与独立队列。
func TestEncodeDecodeGatewayTasksRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		make func(uuid.UUID) (*types.Task, error)
		id   func(*types.Task) uuid.UUID
	}{
		{name: "turn", make: types.NewGatewayTurnTask, id: func(task *types.Task) uuid.UUID { return task.Payload.(*types.GatewayTurnPayload).InboxEventID }},
		{name: "deliver", make: types.NewGatewayDeliverTask, id: func(task *types.Task) uuid.UUID { return task.Payload.(*types.GatewayDeliverPayload).OutboxMessageID }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wantID := uuid.New()
			task, err := test.make(wantID)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := queueredis.EncodeTask(task)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := queueredis.DecodeTask(encoded)
			if err != nil {
				t.Fatal(err)
			}
			if decoded.Queue != types.QueueGateway || test.id(decoded) != wantID {
				t.Fatalf("unexpected gateway task: %#v", decoded)
			}
		})
	}
}

func TestDecodeTaskRejectsInvalidPayload(t *testing.T) {
	// 验证 Redis/Asynq 解码会拒绝非法 JSON 和未知任务，避免错误任务进入 handler。
	badJSON := asynq.NewTask(string(types.TaskParseDocument), []byte("{bad json"))
	if _, err := queueredis.DecodeTask(badJSON); err == nil {
		t.Fatal("DecodeTask bad JSON error = nil, want error")
	}

	unknown := asynq.NewTask("unknown:task", nil)
	if _, err := queueredis.DecodeTask(unknown); err == nil {
		t.Fatal("DecodeTask unknown task error = nil, want error")
	}
}

func TestRouterForwardsDecodedDomainTask(t *testing.T) {
	// 验证 Router adapter 会把 Asynq task 解码成 domain task 再交给业务 handler。
	userID := uuid.New()
	documentID := uuid.New()
	task, err := types.NewParseDocumentTask(userID, documentID)
	if err != nil {
		t.Fatalf("NewParseDocumentTask error = %v", err)
	}
	asynqTask, err := queueredis.EncodeTask(task)
	if err != nil {
		t.Fatalf("EncodeTask error = %v", err)
	}

	var got *types.Task
	mux := asynq.NewServeMux()
	router := queueredis.NewRouter(mux)
	router.Handle(types.TaskParseDocument, queue.HandlerFunc(func(ctx context.Context, task *types.Task) error {
		got = task
		return nil
	}))

	if err := mux.ProcessTask(context.Background(), asynqTask); err != nil {
		t.Fatalf("ProcessTask error = %v", err)
	}
	if got == nil {
		t.Fatal("handler task = nil, want domain task")
	}
	payload, ok := got.Payload.(*types.ParseDocumentPayload)
	if !ok {
		t.Fatalf("payload type = %T, want *types.ParseDocumentPayload", got.Payload)
	}
	if payload.UserID != userID || payload.DocumentID != documentID {
		t.Fatalf("payload = %+v, want user/document ids", payload)
	}
}

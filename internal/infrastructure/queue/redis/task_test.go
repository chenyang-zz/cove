package redis_test

import (
	"context"
	"testing"

	"github.com/boxify/api-go/internal/domain"
	"github.com/boxify/api-go/internal/infrastructure/queue"
	queueredis "github.com/boxify/api-go/internal/infrastructure/queue/redis"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

func TestEncodeDecodeParseDocumentTaskRoundTrip(t *testing.T) {
	// 验证 Redis/Asynq 实现能复用 domain 任务契约完成编码和解码。
	userID := uuid.New()
	documentID := uuid.New()
	task, err := domain.NewParseDocumentTask(userID, documentID)
	if err != nil {
		t.Fatalf("NewParseDocumentTask error = %v", err)
	}

	asynqTask, err := queueredis.EncodeTask(task)
	if err != nil {
		t.Fatalf("EncodeTask error = %v", err)
	}
	if asynqTask.Type() != string(domain.TaskParseDocument) {
		t.Fatalf("task type = %q, want %q", asynqTask.Type(), domain.TaskParseDocument)
	}

	got, err := queueredis.DecodeTask(asynqTask)
	if err != nil {
		t.Fatalf("DecodeTask error = %v", err)
	}
	payload, ok := got.Payload.(*domain.ParseDocumentPayload)
	if !ok {
		t.Fatalf("payload type = %T, want *domain.ParseDocumentPayload", got.Payload)
	}
	if payload.UserID != userID || payload.DocumentID != documentID {
		t.Fatalf("payload = %+v, want user/document ids", payload)
	}
}

func TestDecodeTaskRejectsInvalidPayload(t *testing.T) {
	// 验证 Redis/Asynq 解码会拒绝非法 JSON 和未知任务，避免错误任务进入 handler。
	badJSON := asynq.NewTask(string(domain.TaskParseDocument), []byte("{bad json"))
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
	task, err := domain.NewParseDocumentTask(userID, documentID)
	if err != nil {
		t.Fatalf("NewParseDocumentTask error = %v", err)
	}
	asynqTask, err := queueredis.EncodeTask(task)
	if err != nil {
		t.Fatalf("EncodeTask error = %v", err)
	}

	var got *domain.Task
	mux := asynq.NewServeMux()
	router := queueredis.NewRouter(mux)
	router.Handle(domain.TaskParseDocument, queue.HandlerFunc(func(ctx context.Context, task *domain.Task) error {
		got = task
		return nil
	}))

	if err := mux.ProcessTask(context.Background(), asynqTask); err != nil {
		t.Fatalf("ProcessTask error = %v", err)
	}
	if got == nil {
		t.Fatal("handler task = nil, want domain task")
	}
	payload, ok := got.Payload.(*domain.ParseDocumentPayload)
	if !ok {
		t.Fatalf("payload type = %T, want *domain.ParseDocumentPayload", got.Payload)
	}
	if payload.UserID != userID || payload.DocumentID != documentID {
		t.Fatalf("payload = %+v, want user/document ids", payload)
	}
}

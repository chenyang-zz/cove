package types_test

import (
	"testing"

	"github.com/boxify/api-go/internal/domain/types"
	"github.com/google/uuid"
)

// 验证业务任务名称顺序稳定，worker 注册和调度任务不会因为 map 迭代产生抖动。
func TestTaskNamesAreStable(t *testing.T) {
	names := types.TaskNames()
	want := []types.TaskName{
		types.TaskParseDocument,
		types.TaskParseImage,
		types.TaskMemoryExtract,
		types.TaskMemoryConsolidate,
		types.TaskResearchRun,
		types.TaskGatewayTurn,
		types.TaskGatewayDeliver,
	}
	if len(names) != len(want) {
		t.Fatalf("names = %#v", names)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("names[%d] = %q, want %q", i, names[i], want[i])
		}
	}
}

// 验证网关任务使用独立队列并拒绝空收件箱 ID。
func TestNewGatewayTurnTaskBuildsTypedPayload(t *testing.T) {
	inboxID := uuid.New()
	task, err := types.NewGatewayTurnTask(inboxID)
	if err != nil {
		t.Fatal(err)
	}
	if task.Queue != types.QueueGateway || task.Name != types.TaskGatewayTurn {
		t.Fatalf("unexpected task: %#v", task)
	}
	payload, ok := task.Payload.(*types.GatewayTurnPayload)
	if !ok || payload.InboxEventID != inboxID {
		t.Fatalf("unexpected payload: %#v", task.Payload)
	}
	if _, err := types.NewGatewayTurnTask(uuid.Nil); err == nil {
		t.Fatal("expected nil inbox id error")
	}
}

// 验证文档解析任务由 domain 类型包统一构造，并带上 parse 队列和强类型 payload。
func TestNewParseDocumentTaskBuildsTypedDomainTask(t *testing.T) {
	userID := uuid.New()
	documentID := uuid.New()

	task, err := types.NewParseDocumentTask(userID, documentID)
	if err != nil {
		t.Fatalf("NewParseDocumentTask error = %v", err)
	}
	if task.Name != types.TaskParseDocument {
		t.Fatalf("task name = %q, want %q", task.Name, types.TaskParseDocument)
	}
	if task.Queue != types.QueueParse {
		t.Fatalf("task queue = %q, want %q", task.Queue, types.QueueParse)
	}
	payload, ok := task.Payload.(*types.ParseDocumentPayload)
	if !ok {
		t.Fatalf("payload type = %T, want *types.ParseDocumentPayload", task.Payload)
	}
	if payload.UserID != userID || payload.DocumentID != documentID {
		t.Fatalf("payload = %+v, want user/document ids", payload)
	}
}

// 验证文档解析任务会拒绝空 UUID，避免无效任务进入队列。
func TestNewParseDocumentTaskRejectsNilIDs(t *testing.T) {
	if _, err := types.NewParseDocumentTask(uuid.Nil, uuid.New()); err == nil {
		t.Fatal("NewParseDocumentTask user nil error = nil, want error")
	}
	if _, err := types.NewParseDocumentTask(uuid.New(), uuid.Nil); err == nil {
		t.Fatal("NewParseDocumentTask document nil error = nil, want error")
	}
}

// 验证图片解析任务由 domain 类型包统一构造，并带上 parse 队列和强类型 payload。
func TestNewParseImageTaskBuildsTypedDomainTask(t *testing.T) {
	userID := uuid.New()
	imageID := uuid.New()

	task, err := types.NewParseImageTask(userID, imageID)
	if err != nil {
		t.Fatalf("NewParseImageTask error = %v", err)
	}
	if task.Name != types.TaskParseImage {
		t.Fatalf("task name = %q, want %q", task.Name, types.TaskParseImage)
	}
	if task.Queue != types.QueueParse {
		t.Fatalf("task queue = %q, want %q", task.Queue, types.QueueParse)
	}
	payload, ok := task.Payload.(*types.ParseImagePayload)
	if !ok {
		t.Fatalf("payload type = %T, want *types.ParseImagePayload", task.Payload)
	}
	if payload.UserID != userID || payload.ImageID != imageID {
		t.Fatalf("payload = %+v, want user/image ids", payload)
	}
}

// 验证图片解析任务会拒绝空 UUID，避免无效任务进入队列。
func TestNewParseImageTaskRejectsNilIDs(t *testing.T) {
	if _, err := types.NewParseImageTask(uuid.Nil, uuid.New()); err == nil {
		t.Fatal("NewParseImageTask user nil error = nil, want error")
	}
	if _, err := types.NewParseImageTask(uuid.New(), uuid.Nil); err == nil {
		t.Fatal("NewParseImageTask image nil error = nil, want error")
	}
}

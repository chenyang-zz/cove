package domain_test

import (
	"testing"

	"github.com/boxify/api-go/internal/domain"
	"github.com/google/uuid"
)

func TestTaskNamesAreStable(t *testing.T) {
	// 验证业务任务名称顺序稳定，worker 注册和调度任务不会因为 map 迭代产生抖动。
	names := domain.TaskNames()
	want := []domain.TaskName{
		domain.TaskParseDocument,
		domain.TaskParseImage,
		domain.TaskMemoryExtract,
		domain.TaskMemoryConsolidate,
		domain.TaskResearchRun,
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

func TestNewParseDocumentTaskBuildsTypedDomainTask(t *testing.T) {
	// 验证文档解析任务由 domain 层统一构造，并带上 parse 队列和强类型 payload。
	userID := uuid.New()
	documentID := uuid.New()

	task, err := domain.NewParseDocumentTask(userID, documentID)
	if err != nil {
		t.Fatalf("NewParseDocumentTask error = %v", err)
	}
	if task.Name != domain.TaskParseDocument {
		t.Fatalf("task name = %q, want %q", task.Name, domain.TaskParseDocument)
	}
	if task.Queue != domain.QueueParse {
		t.Fatalf("task queue = %q, want %q", task.Queue, domain.QueueParse)
	}
	payload, ok := task.Payload.(*domain.ParseDocumentPayload)
	if !ok {
		t.Fatalf("payload type = %T, want *domain.ParseDocumentPayload", task.Payload)
	}
	if payload.UserID != userID || payload.DocumentID != documentID {
		t.Fatalf("payload = %+v, want user/document ids", payload)
	}
}

func TestNewParseDocumentTaskRejectsNilIDs(t *testing.T) {
	// 验证文档解析任务会拒绝空 UUID，避免无效任务进入队列。
	if _, err := domain.NewParseDocumentTask(uuid.Nil, uuid.New()); err == nil {
		t.Fatal("NewParseDocumentTask user nil error = nil, want error")
	}
	if _, err := domain.NewParseDocumentTask(uuid.New(), uuid.Nil); err == nil {
		t.Fatal("NewParseDocumentTask document nil error = nil, want error")
	}
}

package postgres_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/repository"
	repositorypostgres "github.com/boxify/api-go/internal/repository/postgres"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

func TestDocumentRepositoryListFiltersAndPaginatesWhenPostgresEnvIsConfigured(t *testing.T) {
	// 验证文档列表在数据库层完成 kb、tag、count、limit/offset 分页，并返回文档关联标签。
	db := newAuthTestDB(t)
	ctx := context.Background()
	userRepo := repositorypostgres.NewUserRepository(db)
	kbRepo := repositorypostgres.NewKnowledgeBaseRepository(db)
	docRepo := repositorypostgres.NewDocumentRepository(db)

	user, err := userRepo.Create(ctx, &models.User{Username: "doc-list-" + uuid.NewString(), PasswordHash: "hash"})
	if err != nil {
		t.Fatalf("Create user error = %v", err)
	}
	kbA, err := kbRepo.Create(ctx, user.ID, &models.KnowledgeBase{Name: "A"})
	if err != nil {
		t.Fatalf("Create kbA error = %v", err)
	}
	kbB, err := kbRepo.Create(ctx, user.ID, &models.KnowledgeBase{Name: "B"})
	if err != nil {
		t.Fatalf("Create kbB error = %v", err)
	}
	tag := &models.Tag{ID: uuid.New(), UserID: user.ID, Name: "重要", Color: "#155EEF"}
	if err := db.WithContext(ctx).Create(tag).Error; err != nil {
		t.Fatalf("Create tag error = %v", err)
	}
	t.Cleanup(func() {
		db.WithContext(context.Background()).Exec("DELETE FROM document_tags WHERE tag_id = ?", tag.ID)
		db.WithContext(context.Background()).Exec("DELETE FROM documents WHERE user_id = ?", user.ID)
		db.WithContext(context.Background()).Exec("DELETE FROM knowledge_bases WHERE user_id = ?", user.ID)
		db.WithContext(context.Background()).Exec("DELETE FROM tags WHERE user_id = ?", user.ID)
		db.WithContext(context.Background()).Exec("DELETE FROM users WHERE id = ?", user.ID)
	})

	doc1, err := docRepo.Create(ctx, user.ID, &models.Document{KBID: &kbA.ID, FileName: "a.txt", FileExt: ".txt", FileSize: 1, FileKey: "a", SourceType: "file", Status: "pending"})
	if err != nil {
		t.Fatalf("Create doc1 error = %v", err)
	}
	doc2, err := docRepo.Create(ctx, user.ID, &models.Document{KBID: &kbA.ID, FileName: "b.txt", FileExt: ".txt", FileSize: 1, FileKey: "b", SourceType: "file", Status: "pending"})
	if err != nil {
		t.Fatalf("Create doc2 error = %v", err)
	}
	if _, err := docRepo.Create(ctx, user.ID, &models.Document{KBID: &kbB.ID, FileName: "c.txt", FileExt: ".txt", FileSize: 1, FileKey: "c", SourceType: "file", Status: "pending"}); err != nil {
		t.Fatalf("Create doc3 error = %v", err)
	}
	if err := db.WithContext(ctx).Exec("INSERT INTO document_tags (document_id, tag_id) VALUES (?, ?), (?, ?)", doc1.ID, tag.ID, doc2.ID, tag.ID).Error; err != nil {
		t.Fatalf("insert document_tags error = %v", err)
	}
	if err := db.WithContext(ctx).Model(&models.Document{}).Where("id = ?", doc1.ID).Update("updated_at", time.Now().Add(-time.Hour)).Error; err != nil {
		t.Fatalf("update doc1 time error = %v", err)
	}
	if err := db.WithContext(ctx).Model(&models.Document{}).Where("id = ?", doc2.ID).Update("updated_at", time.Now()).Error; err != nil {
		t.Fatalf("update doc2 time error = %v", err)
	}

	allRows, err := docRepo.List(ctx, user.ID)
	if err != nil {
		t.Fatalf("List error = %v", err)
	}
	if len(allRows) != 3 {
		t.Fatalf("all rows len = %d, want 3", len(allRows))
	}

	rows, total, err := docRepo.PageList(ctx, user.ID, repository.DocumentListQuery{
		KBID: &kbA.ID,
		Tag:  &tag.Name,
		PageQuery: repository.PageQuery{
			Page:     2,
			PageSize: 1,
		},
	})
	if err != nil {
		t.Fatalf("PageList error = %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	if len(rows) != 1 || rows[0].ID != doc1.ID {
		t.Fatalf("rows = %+v, want second matching document after updated_at desc ordering", rows)
	}
	if len(rows[0].Tags) != 1 || rows[0].Tags[0].Name != tag.Name {
		t.Fatalf("page list tags = %+v, want loaded tag %q", rows[0].Tags, tag.Name)
	}

	found, err := docRepo.FindByID(ctx, user.ID, doc1.ID)
	if err != nil {
		t.Fatalf("FindByID error = %v", err)
	}
	if len(found.Tags) != 1 || found.Tags[0].Name != tag.Name {
		t.Fatalf("find tags = %+v, want loaded tag %q", found.Tags, tag.Name)
	}
}

func TestTagRepositorySyncDocumentTagsWhenPostgresEnvIsConfigured(t *testing.T) {
	// 验证标签仓储会按名称创建或复用当前用户标签，并替换文档标签关联。
	db := newAuthTestDB(t)
	ctx := context.Background()
	userRepo := repositorypostgres.NewUserRepository(db)
	docRepo := repositorypostgres.NewDocumentRepository(db)
	tagRepo := repositorypostgres.NewTagRepository(db)

	user, err := userRepo.Create(ctx, &models.User{Username: "tag-sync-" + uuid.NewString(), PasswordHash: "hash"})
	if err != nil {
		t.Fatalf("Create user error = %v", err)
	}
	otherUser, err := userRepo.Create(ctx, &models.User{Username: "tag-sync-other-" + uuid.NewString(), PasswordHash: "hash"})
	if err != nil {
		t.Fatalf("Create other user error = %v", err)
	}
	doc, err := docRepo.Create(ctx, user.ID, &models.Document{FileName: "a.txt", FileExt: ".txt", FileSize: 1, FileKey: "a", SourceType: "file", Status: "pending"})
	if err != nil {
		t.Fatalf("Create document error = %v", err)
	}
	existing := &models.Tag{ID: uuid.New(), UserID: user.ID, Name: "手动", Color: "#155EEF"}
	otherUserSameName := &models.Tag{ID: uuid.New(), UserID: otherUser.ID, Name: "自动", Color: "#155EEF"}
	if err := db.WithContext(ctx).Create(existing).Error; err != nil {
		t.Fatalf("Create existing tag error = %v", err)
	}
	if err := db.WithContext(ctx).Create(otherUserSameName).Error; err != nil {
		t.Fatalf("Create other user tag error = %v", err)
	}
	t.Cleanup(func() {
		db.WithContext(context.Background()).Exec("DELETE FROM document_tags WHERE document_id = ?", doc.ID)
		db.WithContext(context.Background()).Exec("DELETE FROM documents WHERE user_id IN ?", []uuid.UUID{user.ID, otherUser.ID})
		db.WithContext(context.Background()).Exec("DELETE FROM tags WHERE user_id IN ?", []uuid.UUID{user.ID, otherUser.ID})
		db.WithContext(context.Background()).Exec("DELETE FROM users WHERE id IN ?", []uuid.UUID{user.ID, otherUser.ID})
	})

	rows, err := tagRepo.SyncDocumentTags(ctx, user.ID, doc.ID, []string{" 手动 ", "自动", "手动", ""})
	if err != nil {
		t.Fatalf("SyncDocumentTags error = %v", err)
	}
	if len(rows) != 2 || rows[0].ID != existing.ID || rows[0].Name != "手动" || rows[1].Name != "自动" || rows[1].UserID != user.ID || rows[1].ID == otherUserSameName.ID {
		t.Fatalf("synced rows = %+v, want existing current-user tag and new current-user tag", rows)
	}

	found, err := docRepo.FindByID(ctx, user.ID, doc.ID)
	if err != nil {
		t.Fatalf("FindByID after sync error = %v", err)
	}
	gotNames := documentTagNamesForTest(found.Tags)
	if !slices.Equal(gotNames, []string{"手动", "自动"}) {
		t.Fatalf("document tags = %v, want synced names", gotNames)
	}

	rows, err = tagRepo.SyncDocumentTags(ctx, user.ID, doc.ID, []string{"新标签"})
	if err != nil {
		t.Fatalf("SyncDocumentTags replace error = %v", err)
	}
	if len(rows) != 1 || rows[0].Name != "新标签" {
		t.Fatalf("replaced rows = %+v, want only 新标签", rows)
	}
	found, err = docRepo.FindByID(ctx, user.ID, doc.ID)
	if err != nil {
		t.Fatalf("FindByID after replace error = %v", err)
	}
	gotNames = documentTagNamesForTest(found.Tags)
	if !slices.Equal(gotNames, []string{"新标签"}) {
		t.Fatalf("document tags after replace = %v, want only 新标签", gotNames)
	}

	rows, err = tagRepo.SyncDocumentTags(ctx, user.ID, doc.ID, nil)
	if err != nil {
		t.Fatalf("SyncDocumentTags empty error = %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("empty sync rows = %+v, want empty", rows)
	}
	found, err = docRepo.FindByID(ctx, user.ID, doc.ID)
	if err != nil {
		t.Fatalf("FindByID after empty sync error = %v", err)
	}
	if len(found.Tags) != 0 {
		t.Fatalf("document tags after empty sync = %+v, want empty association", found.Tags)
	}
}

func TestRepositoryCountByKnowledgeBaseWhenPostgresEnvIsConfigured(t *testing.T) {
	// 验证文档和图片仓储会按当前用户、指定知识库批量统计数量，并忽略未归属和其他用户的数据。
	db := newAuthTestDB(t)
	ctx := context.Background()
	userRepo := repositorypostgres.NewUserRepository(db)
	kbRepo := repositorypostgres.NewKnowledgeBaseRepository(db)
	docRepo := repositorypostgres.NewDocumentRepository(db)
	imageRepo := repositorypostgres.NewImageRepository(db)

	user, err := userRepo.Create(ctx, &models.User{Username: "kb-count-" + uuid.NewString(), PasswordHash: "hash"})
	if err != nil {
		t.Fatalf("Create user error = %v", err)
	}
	otherUser, err := userRepo.Create(ctx, &models.User{Username: "kb-count-other-" + uuid.NewString(), PasswordHash: "hash"})
	if err != nil {
		t.Fatalf("Create other user error = %v", err)
	}
	kbA, err := kbRepo.Create(ctx, user.ID, &models.KnowledgeBase{Name: "A"})
	if err != nil {
		t.Fatalf("Create kbA error = %v", err)
	}
	kbB, err := kbRepo.Create(ctx, user.ID, &models.KnowledgeBase{Name: "B"})
	if err != nil {
		t.Fatalf("Create kbB error = %v", err)
	}
	otherKB, err := kbRepo.Create(ctx, otherUser.ID, &models.KnowledgeBase{Name: "Other"})
	if err != nil {
		t.Fatalf("Create other kb error = %v", err)
	}
	t.Cleanup(func() {
		db.WithContext(context.Background()).Exec("DELETE FROM images WHERE user_id IN ?", []uuid.UUID{user.ID, otherUser.ID})
		db.WithContext(context.Background()).Exec("DELETE FROM documents WHERE user_id IN ?", []uuid.UUID{user.ID, otherUser.ID})
		db.WithContext(context.Background()).Exec("DELETE FROM knowledge_bases WHERE user_id IN ?", []uuid.UUID{user.ID, otherUser.ID})
		db.WithContext(context.Background()).Exec("DELETE FROM users WHERE id IN ?", []uuid.UUID{user.ID, otherUser.ID})
	})

	for i, kbID := range []*uuid.UUID{&kbA.ID, &kbA.ID, &kbB.ID, nil} {
		if _, err := docRepo.Create(ctx, user.ID, &models.Document{KBID: kbID, FileName: "doc.txt", FileExt: ".txt", FileSize: int64(i + 1), FileKey: uuid.NewString(), SourceType: "file", Status: "pending"}); err != nil {
			t.Fatalf("Create document %d error = %v", i, err)
		}
	}
	if _, err := docRepo.Create(ctx, otherUser.ID, &models.Document{KBID: &kbA.ID, FileName: "other.txt", FileExt: ".txt", FileSize: 1, FileKey: uuid.NewString(), SourceType: "file", Status: "pending"}); err != nil {
		t.Fatalf("Create other user document error = %v", err)
	}
	if _, err := docRepo.Create(ctx, otherUser.ID, &models.Document{KBID: &otherKB.ID, FileName: "other-kb.txt", FileExt: ".txt", FileSize: 1, FileKey: uuid.NewString(), SourceType: "file", Status: "pending"}); err != nil {
		t.Fatalf("Create other kb document error = %v", err)
	}

	for i, kbID := range []*uuid.UUID{&kbA.ID, &kbB.ID, &kbB.ID, nil} {
		if _, err := imageRepo.Create(ctx, user.ID, &models.Image{KBID: kbID, FileName: "img.png", FileExt: ".png", FileSize: int64(i + 1), FileKey: uuid.NewString(), Status: "pending"}); err != nil {
			t.Fatalf("Create image %d error = %v", i, err)
		}
	}
	if _, err := imageRepo.Create(ctx, otherUser.ID, &models.Image{KBID: &kbB.ID, FileName: "other.png", FileExt: ".png", FileSize: 1, FileKey: uuid.NewString(), Status: "pending"}); err != nil {
		t.Fatalf("Create other user image error = %v", err)
	}

	docCounts, err := docRepo.CountByKnowledgeBase(ctx, user.ID, []uuid.UUID{kbA.ID, kbB.ID, otherKB.ID})
	if err != nil {
		t.Fatalf("Document CountByKnowledgeBase error = %v", err)
	}
	if docCounts[kbA.ID] != 2 || docCounts[kbB.ID] != 1 || docCounts[otherKB.ID] != 0 {
		t.Fatalf("doc counts = %+v, want A=2 B=1 other=0", docCounts)
	}
	imageCounts, err := imageRepo.CountByKnowledgeBase(ctx, user.ID, []uuid.UUID{kbA.ID, kbB.ID, otherKB.ID})
	if err != nil {
		t.Fatalf("Image CountByKnowledgeBase error = %v", err)
	}
	if imageCounts[kbA.ID] != 1 || imageCounts[kbB.ID] != 2 || imageCounts[otherKB.ID] != 0 {
		t.Fatalf("image counts = %+v, want A=1 B=2 other=0", imageCounts)
	}

	emptyDocCounts, err := docRepo.CountByKnowledgeBase(ctx, user.ID, nil)
	if err != nil {
		t.Fatalf("Document CountByKnowledgeBase empty error = %v", err)
	}
	emptyImageCounts, err := imageRepo.CountByKnowledgeBase(ctx, user.ID, []uuid.UUID{})
	if err != nil {
		t.Fatalf("Image CountByKnowledgeBase empty error = %v", err)
	}
	if len(emptyDocCounts) != 0 || len(emptyImageCounts) != 0 {
		t.Fatalf("empty counts = docs:%+v images:%+v, want empty maps", emptyDocCounts, emptyImageCounts)
	}
}

func documentTagNamesForTest(rows []models.Tag) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.Name)
	}
	slices.Sort(out)
	return out
}

func TestKnowledgeBaseRepositoryFindDefaultWhenPostgresEnvIsConfigured(t *testing.T) {
	// 验证知识库仓储可以直接查询当前用户的默认知识库，并在不存在时返回 not found。
	db := newAuthTestDB(t)
	ctx := context.Background()
	userRepo := repositorypostgres.NewUserRepository(db)
	kbRepo := repositorypostgres.NewKnowledgeBaseRepository(db)

	user, err := userRepo.Create(ctx, &models.User{Username: "kb-default-" + uuid.NewString(), PasswordHash: "hash"})
	if err != nil {
		t.Fatalf("Create user error = %v", err)
	}
	otherUser, err := userRepo.Create(ctx, &models.User{Username: "kb-default-other-" + uuid.NewString(), PasswordHash: "hash"})
	if err != nil {
		t.Fatalf("Create other user error = %v", err)
	}
	t.Cleanup(func() {
		db.WithContext(context.Background()).Exec("DELETE FROM knowledge_bases WHERE user_id IN ?", []uuid.UUID{user.ID, otherUser.ID})
		db.WithContext(context.Background()).Exec("DELETE FROM users WHERE id IN ?", []uuid.UUID{user.ID, otherUser.ID})
	})

	if _, err := kbRepo.FindDefault(ctx, user.ID); xerr.From(err).Kind != xerr.KindNotFound {
		t.Fatalf("FindDefault missing error = %v, want not found", err)
	}
	defaultKB, err := kbRepo.Create(ctx, user.ID, &models.KnowledgeBase{Name: "默认知识库", IsDefault: true, ChatEnabled: true})
	if err != nil {
		t.Fatalf("Create default kb error = %v", err)
	}
	if _, err := kbRepo.Create(ctx, otherUser.ID, &models.KnowledgeBase{Name: "其他默认", IsDefault: true, ChatEnabled: true}); err != nil {
		t.Fatalf("Create other default kb error = %v", err)
	}

	found, err := kbRepo.FindDefault(ctx, user.ID)
	if err != nil {
		t.Fatalf("FindDefault error = %v", err)
	}
	if found.ID != defaultKB.ID || found.UserID != user.ID {
		t.Fatalf("FindDefault = %+v, want current user's default", found)
	}
}

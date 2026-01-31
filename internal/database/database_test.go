package database

import (
	"context"
	"os"
	"testing"
	"time"

	"zee-mirror/internal/domain"
)

func setupTestDB(t *testing.T) (*DB, string) {
	tempDir, err := os.MkdirTemp("", "zee-mirror-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	db, err := NewDB(tempDir)
	if err != nil {
		_ = os.RemoveAll(tempDir)
		t.Fatalf("failed to create test db: %v", err)
	}

	return db, tempDir
}

func TestUserOperations(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer func() { _ = os.RemoveAll(tempDir) }()
	defer func() { _ = db.Close() }()

	ctx := context.Background()
	userID := int64(12345)
	username := "testuser"
	role := "authorized"

	err := db.Upsert(ctx, userID, username, role)
	if err != nil {
		t.Errorf("Upsert failed: %v", err)
	}

	gotUsername, gotRole, err := db.GetByID(ctx, userID)
	if err != nil {
		t.Errorf("GetByID failed: %v", err)
	}
	if gotUsername != username || gotRole != role {
		t.Errorf("GetByID got %s, %s; want %s, %s", gotUsername, gotRole, username, role)
	}

	newRole := "admin"
	err = db.SetRole(ctx, userID, newRole)
	if err != nil {
		t.Errorf("SetRole failed: %v", err)
	}

	_, gotRole, _ = db.GetByID(ctx, userID)
	if gotRole != newRole {
		t.Errorf("SetRole failed to update role, got %s want %s", gotRole, newRole)
	}

	count, err := db.GetCount(ctx)
	if err != nil {
		t.Errorf("GetCount failed: %v", err)
	}
	if count != 1 {
		t.Errorf("GetCount got %d, want 1", count)
	}

	users, err := db.GetAll(ctx)
	if err != nil {
		t.Errorf("GetAll failed: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("GetAll got %d users, want 1", len(users))
	}
}

func TestTaskOperations(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer func() { _ = os.RemoveAll(tempDir) }()
	defer func() { _ = db.Close() }()

	ctx := context.Background()
	task := domain.TaskRecord{
		ID:        "task123",
		GID:       "gid123",
		Type:      "mirror",
		Status:    "downloading",
		URL:       "http://example.com/file.zip",
		FileName:  "file.zip",
		ChatID:    111,
		UserID:    222,
		CreatedAt: time.Now(),
	}

	err := db.Save(ctx, task)
	if err != nil {
		t.Errorf("Save task failed: %v", err)
	}

	gotTask, err := db.GetTaskByID(ctx, task.ID)
	if err != nil {
		t.Errorf("GetTaskByID failed: %v", err)
	}
	if gotTask.ID != task.ID || gotTask.Status != task.Status {
		t.Errorf("GetTaskByID returned wrong task: %+v", gotTask)
	}

	newStatus := "completed"
	err = db.UpdateStatus(ctx, task.ID, newStatus, "")
	if err != nil {
		t.Errorf("UpdateStatus failed: %v", err)
	}

	gotTask, _ = db.GetTaskByID(ctx, task.ID)
	if gotTask.Status != newStatus {
		t.Errorf("UpdateStatus failed to update status, got %s want %s", gotTask.Status, newStatus)
	}

	activeTasks, err := db.GetActive(ctx)
	if err != nil {
		t.Errorf("GetActive failed: %v", err)
	}
	if len(activeTasks) != 0 {
		t.Errorf("GetActive got %d tasks, want 0", len(activeTasks))
	}
}

func TestSettingsOperations(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer func() { _ = os.RemoveAll(tempDir) }()
	defer func() { _ = db.Close() }()

	ctx := context.Background()
	key := "test_key"
	val := "test_val"

	err := db.Set(ctx, key, val)
	if err != nil {
		t.Errorf("Set setting failed: %v", err)
	}

	gotVal, err := db.Get(ctx, key)
	if err != nil {
		t.Errorf("Get setting failed: %v", err)
	}
	if gotVal != val {
		t.Errorf("Get setting got %s, want %s", gotVal, val)
	}
}

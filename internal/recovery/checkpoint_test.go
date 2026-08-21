package recovery

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"zee-mirror/internal/database"
	"zee-mirror/internal/domain"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "modernc.org/sqlite"
)

func newTestDB(t *testing.T) *database.DB {
	t.Helper()
	db, err := database.NewDB("sqlite", t.TempDir(), "", "../../migrations")
	if err != nil {
		t.Fatalf("open temp db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestCheckpointLifecycle(t *testing.T) {
	cm := NewCheckpointManager(newTestDB(t))
	task := &domain.Task{ID: "task-1", DownloadedSize: 100, TotalSize: 200, Progress: 50}

	assert.NoError(t, cm.SaveCheckpoint(task))

	cp, err := cm.GetCheckpoint("task-1")
	assert.NoError(t, err)
	if assert.NotNil(t, cp) {
		assert.Equal(t, "task-1", cp.TaskID)
		assert.Equal(t, int64(100), cp.DownloadedBytes)
		assert.Equal(t, int64(200), cp.TotalBytes)
		assert.InDelta(t, 50, cp.Progress, 0.001)
	}

	// Upsert overwrites the previous checkpoint.
	task.DownloadedSize = 150
	assert.NoError(t, cm.SaveCheckpoint(task))
	cp, err = cm.GetCheckpoint("task-1")
	assert.NoError(t, err)
	if assert.NotNil(t, cp) {
		assert.Equal(t, int64(150), cp.DownloadedBytes)
	}

	assert.NoError(t, cm.DeleteCheckpoint("task-1"))
	cp, err = cm.GetCheckpoint("task-1")
	assert.NoError(t, err)
	assert.Nil(t, cp)

	assert.NoError(t, cm.DeleteCheckpoint("never-existed"))
}

func TestStartPeriodicCheckpoint_FinalSaveOnCancel(t *testing.T) {
	cm := NewCheckpointManager(newTestDB(t))
	task := &domain.Task{ID: "task-2", DownloadedSize: 10, TotalSize: 20, Progress: 5}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		cm.StartPeriodicCheckpoint(ctx, task, 10*time.Millisecond)
		close(done)
	}()

	time.Sleep(35 * time.Millisecond) // let at least one tick fire
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("StartPeriodicCheckpoint did not stop after context cancel")
	}

	cp, err := cm.GetCheckpoint("task-2")
	assert.NoError(t, err)
	assert.NotNil(t, cp, "final checkpoint should be saved on shutdown")
}

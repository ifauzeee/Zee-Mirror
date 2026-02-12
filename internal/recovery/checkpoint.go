package recovery

import (
	"context"
	"log/slog"
	"time"

	"zee-mirror/internal/database"
	"zee-mirror/internal/domain"
)

type CheckpointManager struct {
	DB *database.DB
}

func NewCheckpointManager(db *database.DB) *CheckpointManager {
	return &CheckpointManager{
		DB: db,
	}
}

func (cm *CheckpointManager) SaveCheckpoint(task *domain.Task) error {
	cp := domain.TaskCheckpoint{
		TaskID:          task.ID,
		DownloadedBytes: task.DownloadedSize,
		TotalBytes:      task.TotalSize,
		Progress:        task.Progress,
		LastUpdate:      time.Now(),
	}

	return cm.DB.SaveCheckpoint(context.Background(), cp)
}

func (cm *CheckpointManager) GetCheckpoint(taskID string) (*domain.TaskCheckpoint, error) {
	return cm.DB.GetCheckpoint(context.Background(), taskID)
}

func (cm *CheckpointManager) DeleteCheckpoint(taskID string) error {
	return cm.DB.DeleteCheckpoint(context.Background(), taskID)
}

func (cm *CheckpointManager) StartPeriodicCheckpoint(ctx context.Context, task *domain.Task, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if err := cm.SaveCheckpoint(task); err != nil {
				slog.Error("Failed to save final checkpoint", "taskID", task.ID, "error", err)
			}
			return
		case <-ticker.C:
			if err := cm.SaveCheckpoint(task); err != nil {
				slog.Error("Failed to save checkpoint", "taskID", task.ID, "error", err)
			}
		}
	}
}

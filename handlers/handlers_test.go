package handlers

import (
	"testing"
	"zee-mirror/internal/config"
	"zee-mirror/internal/domain"
	"zee-mirror/internal/repository/mocks"

	"zee-mirror/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTaskManager_CreateTask_Duplicate(t *testing.T) {
	mockRepo := new(mocks.MockRepository)
	tm := &service.TaskManager{
		DB:            mockRepo,
		Config:        &config.Config{},
		Tasks:         make(map[string]*service.Task),
		StopDuplicate: true,
	}

	taskID := "test_task_123"
	existingTask := &service.Task{
		Task: domain.Task{
			ID:     taskID,
			URL:    "http://example.com/file.zip",
			Status: domain.StatusDownloading,
		},
	}
	tm.Tasks[taskID] = existingTask

	mockRepo.On("GetCompletedTaskByURL", mock.Anything, "http://example.com/file.zip", mock.Anything).Return(nil, nil)
	mockRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

	_, err := tm.CreateTask(service.TypeMirror, "http://example.com/file.zip", "file.zip", 123, 456, 0, 789, false, false, "", "", 0, "", false)

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrDuplicateTask)
	assert.Contains(t, err.Error(), taskID)
}

func TestTaskManager_CreateTask_DBDuplicate(t *testing.T) {
	mockRepo := new(mocks.MockRepository)
	tm := &service.TaskManager{
		DB:            mockRepo,
		Config:        &config.Config{},
		Tasks:         make(map[string]*service.Task),
		StopDuplicate: true,
	}

	mockRepo.On("GetCompletedTaskByURL", mock.Anything, "http://example.com/exists.zip", mock.Anything).Return(&domain.TaskRecord{
		ID: "existing_db_id",
	}, nil)

	_, err := tm.CreateTask(service.TypeMirror, "http://example.com/exists.zip", "exists.zip", 123, 456, 0, 789, false, false, "", "", 0, "", false)

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrDuplicateTask)
	assert.Contains(t, err.Error(), "cloud/database")
}

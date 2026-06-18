package service_test

import (
	"testing"

	"zee-mirror/internal/config"
	"zee-mirror/internal/domain"
	"zee-mirror/internal/queue"
	"zee-mirror/internal/repository/mocks"
	"zee-mirror/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestTaskManager(t *testing.T) (*service.TaskManager, *mocks.MockRepository) {
	t.Helper()
	mockRepo := new(mocks.MockRepository)
	mockRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Maybe()
	mockRepo.On("GetActive", mock.Anything).Return([]domain.TaskRecord{}, nil).Maybe()
	mockRepo.On("GetCompletedTaskByURL", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Maybe()

	tm := &service.TaskManager{
		Tasks:       make(map[string]*service.Task),
		Queue:       queue.NewPriorityQueue(),
		QueueSignal: make(chan struct{}, 1000),
		RateLimiter: queue.NewUserRateLimiter(10, 60),
		Config:      &config.Config{MaxRetries: 3, OwnerID: 123},
		DB:          mockRepo,
	}
	return tm, mockRepo
}

func createTestTask(t *testing.T, tm *service.TaskManager, url string) *service.Task {
	t.Helper()
	task, err := tm.CreateTask(
		service.TypeMirror, url, "file.zip",
		12345, 1, 0, 123,
		false, false, "", "", 0, "", false,
	)
	assert.NoError(t, err)
	assert.NotNil(t, task)
	return task
}

func TestTaskManager_GetTask(t *testing.T) {
	t.Run("ExistingTask", func(t *testing.T) {
		tm, _ := newTestTaskManager(t)
		task := createTestTask(t, tm, "http://example.com/file.zip")

		found := tm.GetTask(task.ID)
		assert.NotNil(t, found)
		assert.Equal(t, task.ID, found.ID)
	})

	t.Run("NonExistentTask", func(t *testing.T) {
		tm, _ := newTestTaskManager(t)

		found := tm.GetTask("nonexistent-id")
		assert.Nil(t, found)
	})
}

func TestTaskManager_GetActiveTasks(t *testing.T) {
	t.Run("NoTasks", func(t *testing.T) {
		tm, _ := newTestTaskManager(t)

		tasks := tm.GetActiveTasks()
		assert.Empty(t, tasks)
	})

	t.Run("WithActiveTasks", func(t *testing.T) {
		tm, _ := newTestTaskManager(t)
		task1 := createTestTask(t, tm, "http://example.com/file1.zip")
		task2 := createTestTask(t, tm, "http://example.com/file2.zip")

		tasks := tm.GetActiveTasks()
		assert.Len(t, tasks, 2)

		ids := make(map[string]bool)
		for _, t := range tasks {
			ids[t.ID] = true
		}
		assert.True(t, ids[task1.ID])
		assert.True(t, ids[task2.ID])
	})

	t.Run("ExcludesCompletedTasks", func(t *testing.T) {
		tm, _ := newTestTaskManager(t)
		task1 := createTestTask(t, tm, "http://example.com/file1.zip")
		_ = createTestTask(t, tm, "http://example.com/file2.zip")

		task1.SetStatus(service.StatusCompleted)

		tasks := tm.GetActiveTasks()
		assert.Len(t, tasks, 1)
	})

	t.Run("ExcludesFailedTasks", func(t *testing.T) {
		tm, _ := newTestTaskManager(t)
		task1 := createTestTask(t, tm, "http://example.com/file1.zip")
		_ = createTestTask(t, tm, "http://example.com/file2.zip")

		task1.SetStatus(service.StatusFailed)

		tasks := tm.GetActiveTasks()
		assert.Len(t, tasks, 1)
	})

	t.Run("ExcludesCancelledTasks", func(t *testing.T) {
		tm, _ := newTestTaskManager(t)
		task1 := createTestTask(t, tm, "http://example.com/file1.zip")
		_ = createTestTask(t, tm, "http://example.com/file2.zip")

		task1.SetStatus(service.StatusCancelled)

		tasks := tm.GetActiveTasks()
		assert.Len(t, tasks, 1)
	})
}

func TestTaskManager_CancelTask(t *testing.T) {
	t.Run("CancelExistingTask", func(t *testing.T) {
		tm, _ := newTestTaskManager(t)
		task := createTestTask(t, tm, "http://example.com/file.zip")

		result := tm.CancelTask(task.ID)
		assert.True(t, result)
		assert.Equal(t, service.StatusCancelled, task.Status)
	})

	t.Run("CancelNonExistentTask", func(t *testing.T) {
		tm, _ := newTestTaskManager(t)

		result := tm.CancelTask("nonexistent-id")
		assert.False(t, result)
	})

	t.Run("CancelAlreadyCompletedTask", func(t *testing.T) {
		tm, _ := newTestTaskManager(t)
		task := createTestTask(t, tm, "http://example.com/file.zip")
		task.SetStatus(service.StatusCompleted)

		result := tm.CancelTask(task.ID)
		assert.False(t, result)
	})

	t.Run("CancelAlreadyCancelledTask", func(t *testing.T) {
		tm, _ := newTestTaskManager(t)
		task := createTestTask(t, tm, "http://example.com/file.zip")
		task.SetStatus(service.StatusCancelled)

		result := tm.CancelTask(task.ID)
		assert.False(t, result)
	})
}

func TestTaskManager_CancelAllTasks(t *testing.T) {
	t.Run("CancelMultipleTasks", func(t *testing.T) {
		tm, _ := newTestTaskManager(t)
		_ = createTestTask(t, tm, "http://example.com/file1.zip")
		_ = createTestTask(t, tm, "http://example.com/file2.zip")
		_ = createTestTask(t, tm, "http://example.com/file3.zip")

		count := tm.CancelAllTasks()
		assert.Equal(t, 3, count)

		for _, task := range tm.Tasks {
			assert.Equal(t, service.StatusCancelled, task.Status)
		}
	})

	t.Run("CancelEmptyTaskList", func(t *testing.T) {
		tm, _ := newTestTaskManager(t)

		count := tm.CancelAllTasks()
		assert.Equal(t, 0, count)
	})

	t.Run("SkipsCompletedTasks", func(t *testing.T) {
		tm, _ := newTestTaskManager(t)
		task1 := createTestTask(t, tm, "http://example.com/file1.zip")
		_ = createTestTask(t, tm, "http://example.com/file2.zip")

		task1.SetStatus(service.StatusCompleted)

		count := tm.CancelAllTasks()
		assert.Equal(t, 1, count)
	})
}

func TestTask_SetStatus(t *testing.T) {
	task := &service.Task{
		Task: domain.Task{
			ID:     "test-status",
			Status: service.StatusQueued,
		},
	}

	task.SetStatus(service.StatusDownloading)
	assert.Equal(t, service.StatusDownloading, task.Status)

	task.SetStatus(service.StatusUploading)
	assert.Equal(t, service.StatusUploading, task.Status)

	task.SetStatus(service.StatusCompleted)
	assert.Equal(t, service.StatusCompleted, task.Status)
}

func TestTask_SetError(t *testing.T) {
	task := &service.Task{
		Task: domain.Task{
			ID:     "test-error",
			Status: service.StatusDownloading,
		},
	}

	task.SetError("download failed")
	assert.Equal(t, "download failed", task.Error)
	assert.Equal(t, service.StatusFailed, task.Status)
}

func TestTask_Cancel(t *testing.T) {
	t.Run("CancelQueuedTask", func(t *testing.T) {
		task := &service.Task{
			Task: domain.Task{
				ID:     "test-cancel",
				Status: service.StatusQueued,
			},
		}

		result := task.Cancel(service.StatusCancelled)
		assert.True(t, result)
		assert.Equal(t, service.StatusCancelled, task.Status)
	})

	t.Run("CancelDownloadingTask", func(t *testing.T) {
		task := &service.Task{
			Task: domain.Task{
				ID:     "test-cancel",
				Status: service.StatusDownloading,
			},
		}

		result := task.Cancel(service.StatusCancelled)
		assert.True(t, result)
		assert.Equal(t, service.StatusCancelled, task.Status)
	})

	t.Run("CancelCompletedTask", func(t *testing.T) {
		task := &service.Task{
			Task: domain.Task{
				ID:     "test-cancel",
				Status: service.StatusCompleted,
			},
		}

		result := task.Cancel(service.StatusCancelled)
		assert.False(t, result)
		assert.Equal(t, service.StatusCompleted, task.Status)
	})
}

func TestTaskManager_CreateTask_DifferentTypes(t *testing.T) {
	types := []struct {
		name     string
		taskType service.TaskType
		url      string
	}{
		{"Mirror", service.TypeMirror, "http://example.com/file.zip"},
		{"Leech", service.TypeLeech, "http://example.com/file.zip"},
		{"YTDLP", service.TypeYTDLP, "http://youtube.com/watch?v=abc"},
		{"Torrent", service.TypeTorrent, "magnet:?xt=urn:btih:abc123"},
		{"Clone", service.TypeClone, "https://drive.google.com/file/d/abc"},
		{"Viking", service.TypeViking, "http://example.com/file.zip"},
	}

	for _, tc := range types {
		t.Run(tc.name, func(t *testing.T) {
			tm, _ := newTestTaskManager(t)
			task, err := tm.CreateTask(
				tc.taskType, tc.url, "file.zip",
				12345, 1, 0, 123,
				false, false, "", "", 0, "", false,
			)
			assert.NoError(t, err)
			assert.NotNil(t, task)
			assert.Equal(t, tc.taskType, task.Type)
			assert.Equal(t, tc.url, task.URL)
		})
	}
}

func TestTaskManager_CreateTask_WithFlags(t *testing.T) {
	t.Run("WithZip", func(t *testing.T) {
		tm, _ := newTestTaskManager(t)
		task, err := tm.CreateTask(
			service.TypeMirror, "http://example.com/file.zip", "file.zip",
			12345, 1, 0, 123,
			true, false, "mypassword", "", 0, "", false,
		)
		assert.NoError(t, err)
		assert.True(t, task.Zip)
		assert.Equal(t, "mypassword", task.Password)
	})

	t.Run("WithUnzip", func(t *testing.T) {
		tm, _ := newTestTaskManager(t)
		task, err := tm.CreateTask(
			service.TypeMirror, "http://example.com/file.zip", "file.zip",
			12345, 1, 0, 123,
			false, true, "", "", 0, "", false,
		)
		assert.NoError(t, err)
		assert.True(t, task.Unzip)
	})

	t.Run("WithQuality", func(t *testing.T) {
		tm, _ := newTestTaskManager(t)
		task, err := tm.CreateTask(
			service.TypeYTDLP, "http://youtube.com/watch?v=abc", "video.mp4",
			12345, 1, 0, 123,
			false, false, "", "1080p", 0, "", false,
		)
		assert.NoError(t, err)
		assert.Equal(t, "1080p", task.Quality)
	})
}

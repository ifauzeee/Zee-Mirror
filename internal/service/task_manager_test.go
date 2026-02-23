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

func TestCreateTask_Success(t *testing.T) {
	mockRepo := new(mocks.MockRepository)

	mockRepo.On("Save", mock.Anything, mock.AnythingOfType("domain.TaskRecord")).Return(nil)

	tm := &service.TaskManager{
		Tasks:       make(map[string]*service.Task),
		Queue:       queue.NewPriorityQueue(),
		QueueSignal: make(chan struct{}, 1000),
		RateLimiter: queue.NewUserRateLimiter(5, 60),
		Config:      &config.Config{MaxRetries: 3, OwnerID: 123},
		DB:          mockRepo,
	}

	task, err := tm.CreateTask(
		service.TypeMirror,
		"http://example.com/file.zip",
		"file.zip",
		12345,
		1,
		0,
		123,
		false, false, "", "", 0, "", false,
	)

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, "http://example.com/file.zip", task.URL)
	assert.Equal(t, service.StatusQueued, task.Status)

	assert.Contains(t, tm.Tasks, task.ID)

	select {
	case <-tm.QueueSignal:

	default:
		t.Error("Task was not enqueued (no signal received)")
	}

	mockRepo.AssertExpectations(t)
}

func TestCreateTask_RateLimit(t *testing.T) {
	mockRepo := new(mocks.MockRepository)

	limiter := queue.NewUserRateLimiter(1, 1)

	limiter.Allow(456)

	tm := &service.TaskManager{
		Tasks:       make(map[string]*service.Task),
		RateLimiter: limiter,
		Config:      &config.Config{MaxRetries: 3},
		DB:          mockRepo,
	}

	task, err := tm.CreateTask(
		service.TypeMirror,
		"http://example.com/file.zip",
		"file.zip",
		12345, 1, 0,
		456,
		false, false, "", "", 0, "", false,
	)

	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "limit")
	assert.ErrorIs(t, err, domain.ErrLimitExceeded)
}

func TestCreateTask_Duplicate(t *testing.T) {
	mockRepo := new(mocks.MockRepository)
	mockRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

	mockRepo.On("GetCompletedTaskByURL", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Maybe()

	tm := &service.TaskManager{
		Tasks:         make(map[string]*service.Task),
		Queue:         queue.NewPriorityQueue(),
		QueueSignal:   make(chan struct{}, 1000),
		RateLimiter:   queue.NewUserRateLimiter(10, 60),
		Config:        &config.Config{MaxRetries: 3},
		DB:            mockRepo,
		StopDuplicate: true,
	}

	url := "http://example.com/duplicate.zip"

	task1, err1 := tm.CreateTask(
		service.TypeMirror, url, "duplicate.zip",
		12345, 1, 0, 789,
		false, false, "", "", 0, "", false,
	)
	assert.NoError(t, err1)

	task1.Status = service.StatusDownloading
	t.Logf("Task1 Status: %s, URL: %s", task1.Status, task1.URL)
	t.Logf("Task1 Finished? %v", task1.Status == service.StatusCompleted || task1.Status == service.StatusFailed || task1.Status == service.StatusCancelled)
	t.Logf("Tasks in map: %d", len(tm.Tasks))

	task2, err2 := tm.CreateTask(
		service.TypeMirror, url, "duplicate.zip",
		12345, 2, 0, 789,
		false, false, "", "", 0, "", false,
	)

	assert.Error(t, err2)
	assert.Nil(t, task2)
	assert.ErrorIs(t, err2, domain.ErrDuplicateTask)
}

package service

import (
	"context"
	"log"
	"time"

	"zee-mirror/internal/domain"
	"zee-mirror/internal/repository"
)

const schedulerPollInterval = 30 * time.Second

type Scheduler struct {
	repo    repository.TaskRepository
	manager *TaskManager
}

func NewScheduler(repo repository.TaskRepository, manager *TaskManager) *Scheduler {
	return &Scheduler{repo: repo, manager: manager}
}

func (s *Scheduler) Start(ctx context.Context) {
	go s.run(ctx)
}

func (s *Scheduler) run(ctx context.Context) {
	ticker := time.NewTicker(schedulerPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.processDue(ctx)
		}
	}
}

func (s *Scheduler) processDue(ctx context.Context) {
	tasks, err := s.repo.GetPendingScheduled(ctx)
	if err != nil {
		log.Printf("scheduler: get pending: %v", err)
		return
	}

	for _, st := range tasks {
		if err := s.executeScheduled(ctx, st); err != nil {
			log.Printf("scheduler: execute %s: %v", st.ID, err)
			continue
		}
	}
}

func (s *Scheduler) executeScheduled(ctx context.Context, st domain.ScheduledTask) error {
	task, err := s.manager.CreateTask(
		domain.TaskType(st.TaskType),
		st.URL,
		st.FileName,
		st.ChatID,
		0,
		0,
		st.UserID,
		st.Zip,
		st.Unzip,
		st.Password,
		st.Quality,
		0,
		"",
		false,
	)
	if err != nil {
		return err
	}

	return s.repo.MarkScheduledDone(ctx, st.ID, task.ID)
}

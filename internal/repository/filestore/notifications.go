package filestore

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/SolaTyolo/herald/internal/domain"
)

func (s *Store) notificationsDir() string {
	return s.collection("notifications")
}

func (s *Store) notificationPath(id string) string {
	return filepath.Join(s.notificationsDir(), id+".json")
}

func (s *Store) jobsDir() string {
	return s.collection("jobs")
}

func (s *Store) jobPath(id string) string {
	return filepath.Join(s.jobsDir(), id+".json")
}

func (s *Store) GetNotification(ctx context.Context, id string) (*domain.Notification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var n domain.Notification
	if err := readJSON(s.notificationPath(id), &n); err != nil {
		return nil, err
	}
	return &n, nil
}

func (s *Store) GetJobsByNotification(ctx context.Context, notificationID string) ([]domain.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs, err := s.listJobsLocked(func(j *domain.Job) bool {
		return j.NotificationID == notificationID
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].StepOrder < jobs[j].StepOrder
	})
	return jobs, nil
}

func (s *Store) CreateNotification(ctx context.Context, n *domain.Notification, jobs []domain.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	n.ID = uuid.NewString()
	n.CreatedAt = time.Now()
	if n.Status == "" {
		n.Status = domain.NotificationPending
	}
	if err := writeJSON(s.notificationPath(n.ID), n); err != nil {
		return err
	}
	now := time.Now()
	for i := range jobs {
		j := &jobs[i]
		j.ID = uuid.NewString()
		j.NotificationID = n.ID
		j.Status = domain.JobPending
		j.CreatedAt = now
		j.UpdatedAt = now
		if err := writeJSON(s.jobPath(j.ID), j); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) GetNotificationByTransaction(ctx context.Context, transactionID, subscriberPK string) (*domain.Notification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := listJSONFiles(s.notificationsDir())
	if err != nil {
		return nil, err
	}
	for _, path := range files {
		var n domain.Notification
		if err := readJSON(path, &n); err != nil {
			return nil, err
		}
		if n.TransactionID == transactionID && n.SubscriberPK == subscriberPK {
			return &n, nil
		}
	}
	return nil, ErrNotFound
}

func (s *Store) ListNotifications(ctx context.Context, limit, offset int) ([]domain.Notification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := listJSONFiles(s.notificationsDir())
	if err != nil {
		return nil, err
	}
	out := make([]domain.Notification, 0, len(files))
	for _, path := range files {
		var n domain.Notification
		if err := readJSON(path, &n); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	out = paginate(out, limit, offset)
	return out, nil
}

func (s *Store) UpdateNotificationStatus(ctx context.Context, id string, status domain.NotificationStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var n domain.Notification
	if err := readJSON(s.notificationPath(id), &n); err != nil {
		return err
	}
	n.Status = status
	return writeJSON(s.notificationPath(n.ID), &n)
}

func (s *Store) GetJob(ctx context.Context, id string) (*domain.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var j domain.Job
	if err := readJSON(s.jobPath(id), &j); err != nil {
		return nil, err
	}
	return &j, nil
}

func (s *Store) GetNextJob(ctx context.Context, notificationID string, afterOrder int) (*domain.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs, err := s.listJobsLocked(func(j *domain.Job) bool {
		return j.NotificationID == notificationID && j.StepOrder > afterOrder
	})
	if err != nil {
		return nil, err
	}
	if len(jobs) == 0 {
		return nil, ErrNotFound
	}
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].StepOrder < jobs[j].StepOrder
	})
	return &jobs[0], nil
}

func (s *Store) UpdateJob(ctx context.Context, job *domain.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(s.jobPath(job.ID)); err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return err
	}
	job.UpdatedAt = time.Now()
	return writeJSON(s.jobPath(job.ID), job)
}

func (s *Store) ListJobsByTransaction(ctx context.Context, transactionID string) ([]domain.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := listJSONFiles(s.notificationsDir())
	if err != nil {
		return nil, err
	}
	notificationIDs := make(map[string]struct{})
	for _, path := range files {
		var n domain.Notification
		if err := readJSON(path, &n); err != nil {
			return nil, err
		}
		if n.TransactionID == transactionID {
			notificationIDs[n.ID] = struct{}{}
		}
	}
	jobs, err := s.listJobsLocked(func(j *domain.Job) bool {
		_, ok := notificationIDs[j.NotificationID]
		return ok
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].StepOrder < jobs[j].StepOrder
	})
	return jobs, nil
}

func (s *Store) listJobsLocked(match func(*domain.Job) bool) ([]domain.Job, error) {
	files, err := listJSONFiles(s.jobsDir())
	if err != nil {
		return nil, err
	}
	var out []domain.Job
	for _, path := range files {
		var j domain.Job
		if err := readJSON(path, &j); err != nil {
			return nil, err
		}
		if match == nil || match(&j) {
			out = append(out, j)
		}
	}
	return out, nil
}

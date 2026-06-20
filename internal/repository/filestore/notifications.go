package filestore

import (
	"context"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/SolaTyolo/herald/internal/domain"
)

func (s *Store) notificationsDir(envID string) string {
	return s.collection(envID, "notifications")
}

func (s *Store) notificationPath(envID, id string) string {
	return filepath.Join(s.notificationsDir(envID), id+".json")
}

func (s *Store) jobsDir(envID string) string {
	return s.collection(envID, "jobs")
}

func (s *Store) jobPath(envID, id string) string {
	return filepath.Join(s.jobsDir(envID), id+".json")
}

func (s *Store) GetNotification(ctx context.Context, id string) (*domain.Notification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.findNotificationByIDLocked(id)
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
	if err := writeJSON(s.notificationPath(n.EnvID, n.ID), n); err != nil {
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
		if err := writeJSON(s.jobPath(n.EnvID, j.ID), j); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) GetNotificationByTransaction(ctx context.Context, envID, transactionID, subscriberPK string) (*domain.Notification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := listJSONFiles(s.notificationsDir(envID))
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

func (s *Store) ListNotifications(ctx context.Context, envID string, limit, offset int) ([]domain.Notification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := listJSONFiles(s.notificationsDir(envID))
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

	n, envID, err := s.findNotificationWithEnvLocked(id)
	if err != nil {
		return err
	}
	n.Status = status
	return writeJSON(s.notificationPath(envID, n.ID), n)
}

func (s *Store) GetJob(ctx context.Context, id string) (*domain.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.findJobByIDLocked(id)
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

	_, envID, err := s.findJobWithEnvLocked(job.ID)
	if err != nil {
		return err
	}
	job.UpdatedAt = time.Now()
	return writeJSON(s.jobPath(envID, job.ID), job)
}

func (s *Store) ListJobsByTransaction(ctx context.Context, envID, transactionID string) ([]domain.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := listJSONFiles(s.notificationsDir(envID))
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
	jobs, err := s.listJobsInEnvLocked(envID, func(j *domain.Job) bool {
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

func (s *Store) findNotificationByIDLocked(id string) (*domain.Notification, error) {
	envIDs, err := s.listEnvIDs()
	if err != nil {
		return nil, err
	}
	for _, envID := range envIDs {
		path := s.notificationPath(envID, id)
		var n domain.Notification
		if err := readJSON(path, &n); err != nil {
			if isNotFound(err) {
				continue
			}
			return nil, err
		}
		return &n, nil
	}
	return nil, ErrNotFound
}

func (s *Store) findNotificationWithEnvLocked(id string) (*domain.Notification, string, error) {
	envIDs, err := s.listEnvIDs()
	if err != nil {
		return nil, "", err
	}
	for _, envID := range envIDs {
		path := s.notificationPath(envID, id)
		var n domain.Notification
		if err := readJSON(path, &n); err != nil {
			if isNotFound(err) {
				continue
			}
			return nil, "", err
		}
		return &n, envID, nil
	}
	return nil, "", ErrNotFound
}

func (s *Store) findJobByIDLocked(id string) (*domain.Job, error) {
	job, _, err := s.findJobWithEnvLocked(id)
	return job, err
}

func (s *Store) findJobWithEnvLocked(id string) (*domain.Job, string, error) {
	envIDs, err := s.listEnvIDs()
	if err != nil {
		return nil, "", err
	}
	for _, envID := range envIDs {
		path := s.jobPath(envID, id)
		var j domain.Job
		if err := readJSON(path, &j); err != nil {
			if isNotFound(err) {
				continue
			}
			return nil, "", err
		}
		return &j, envID, nil
	}
	return nil, "", ErrNotFound
}

func (s *Store) listJobsLocked(match func(*domain.Job) bool) ([]domain.Job, error) {
	envIDs, err := s.listEnvIDs()
	if err != nil {
		return nil, err
	}
	var out []domain.Job
	for _, envID := range envIDs {
		jobs, err := s.listJobsInEnvLocked(envID, match)
		if err != nil {
			return nil, err
		}
		out = append(out, jobs...)
	}
	return out, nil
}

func (s *Store) listJobsInEnvLocked(envID string, match func(*domain.Job) bool) ([]domain.Job, error) {
	files, err := listJSONFiles(s.jobsDir(envID))
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

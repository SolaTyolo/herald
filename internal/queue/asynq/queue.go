package asynqqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

const TypeProcessJob = "workflow:process_job"
const TypeDigestFlush = "workflow:digest_flush"

type Client struct {
	client *asynq.Client
}

func NewClient(redisOpt asynq.RedisClientOpt) *Client {
	return &Client{client: asynq.NewClient(redisOpt)}
}

func (c *Client) Close() error { return c.client.Close() }

type JobPayload struct {
	JobID string `json:"jobId"`
	EnvID string `json:"envId"`
}

type DigestPayload struct {
	BucketID string `json:"bucketId"`
	JobID    string `json:"jobId"`
	EnvID    string `json:"envId"`
}

func (c *Client) EnqueueJob(ctx context.Context, envID, jobID string, processAt *time.Time) error {
	payload, _ := json.Marshal(JobPayload{JobID: jobID, EnvID: envID})
	task := asynq.NewTask(TypeProcessJob, payload)
	var opts []asynq.Option
	opts = append(opts, asynq.MaxRetry(5))
	if processAt != nil {
		opts = append(opts, asynq.ProcessAt(*processAt))
	}
	_, err := c.client.EnqueueContext(ctx, task, opts...)
	return err
}

func (c *Client) EnqueueDigestFlush(ctx context.Context, envID, bucketID, jobID string, at time.Time) error {
	payload, _ := json.Marshal(DigestPayload{BucketID: bucketID, JobID: jobID, EnvID: envID})
	task := asynq.NewTask(TypeDigestFlush, payload)
	_, err := c.client.EnqueueContext(ctx, task, asynq.ProcessAt(at), asynq.MaxRetry(5))
	return err
}

type Server struct {
	server *asynq.Server
	mux    *asynq.ServeMux
}

func NewServer(redisOpt asynq.RedisClientOpt, concurrency int) *Server {
	if concurrency <= 0 {
		concurrency = 10
	}
	srv := asynq.NewServer(redisOpt, asynq.Config{Concurrency: concurrency})
	return &Server{server: srv, mux: asynq.NewServeMux()}
}

func (s *Server) Handle(pattern string, handler func(context.Context, *asynq.Task) error) {
	s.mux.HandleFunc(pattern, handler)
}

func (s *Server) Run() error {
	return s.server.Run(s.mux)
}

func (s *Server) Shutdown() {
	s.server.Shutdown()
}

func ParseJobPayload(task *asynq.Task) (JobPayload, error) {
	var p JobPayload
	if err := json.Unmarshal(task.Payload(), &p); err != nil {
		return p, err
	}
	if p.JobID == "" {
		return p, fmt.Errorf("missing jobId")
	}
	return p, nil
}

func ParseDigestPayload(task *asynq.Task) (DigestPayload, error) {
	var p DigestPayload
	if err := json.Unmarshal(task.Payload(), &p); err != nil {
		return p, err
	}
	return p, nil
}

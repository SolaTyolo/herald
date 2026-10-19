package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/SolaTyolo/herald/internal/bootstrap"
	"github.com/SolaTyolo/herald/internal/delivery"
	asynqqueue "github.com/SolaTyolo/herald/internal/queue/asynq"
)

func main() {
	ctx := context.Background()
	app, err := bootstrap.New(ctx, bootstrap.RoleWorker)
	if err != nil {
		slog.Error("bootstrap failed", "err", err)
		os.Exit(1)
	}
	defer app.Close()

	if !app.Config.UsesAsynq() {
		slog.Error("cmd/worker only runs when QUEUE_MODE=asynq")
		os.Exit(1)
	}

	tracer := otel.Tracer("herald-worker")
	redisOpt := asynq.RedisClientOpt{Addr: app.Config.RedisAddr}
	srv := asynqqueue.NewServer(redisOpt, 20)

	srv.Handle(asynqqueue.TypeProcessJob, func(ctx context.Context, task *asynq.Task) error {
		ctx, span := tracer.Start(ctx, "asynq.process_job")
		defer span.End()
		p, err := asynqqueue.ParseJobPayload(task)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return fmt.Errorf("%w: %w", err, asynq.SkipRetry)
		}
		span.SetAttributes(
			attribute.String("herald.env_id", p.EnvID),
			attribute.String("herald.job_id", p.JobID),
		)
		err = mapWorkerErr(app.Engine.ProcessJob(ctx, p.EnvID, p.JobID))
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	})

	srv.Handle(asynqqueue.TypeDigestFlush, func(ctx context.Context, task *asynq.Task) error {
		ctx, span := tracer.Start(ctx, "asynq.digest_flush")
		defer span.End()
		p, err := asynqqueue.ParseDigestPayload(task)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return fmt.Errorf("%w: %w", err, asynq.SkipRetry)
		}
		span.SetAttributes(
			attribute.String("herald.env_id", p.EnvID),
			attribute.String("herald.job_id", p.JobID),
			attribute.String("herald.bucket_id", p.BucketID),
		)
		err = mapWorkerErr(app.Engine.FlushDigest(ctx, p.EnvID, p.BucketID, p.JobID))
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	})

	srv.Handle(asynqqueue.TypeTopicFanout, func(ctx context.Context, task *asynq.Task) error {
		ctx, span := tracer.Start(ctx, "asynq.topic_fanout")
		defer span.End()
		p, err := asynqqueue.ParseTopicFanoutPayload(task)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return fmt.Errorf("%w: %w", err, asynq.SkipRetry)
		}
		span.SetAttributes(
			attribute.String("herald.env_id", p.EnvID),
			attribute.String("herald.transaction_id", p.TransactionID),
			attribute.String("herald.topic_key", p.TopicKey),
			attribute.String("herald.workflow_id", p.WorkflowID),
		)
		err = mapWorkerErr(app.Trigger.ProcessTopicFanout(ctx, p))
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	})

	go func() {
		slog.Info("worker started")
		if err := srv.Run(); err != nil {
			slog.Error("worker error", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	srv.Shutdown()
}

// mapWorkerErr retries only delivery.RetryableError; permanent failures skip Asynq retry.
func mapWorkerErr(err error) error {
	if err == nil {
		return nil
	}
	if delivery.IsRetryable(err) {
		return err
	}
	return fmt.Errorf("%w: %w", err, asynq.SkipRetry)
}

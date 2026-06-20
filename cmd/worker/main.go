package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"

	"github.com/SolaTyolo/herald/internal/bootstrap"
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

	redisOpt := asynq.RedisClientOpt{Addr: app.Config.RedisAddr}
	srv := asynqqueue.NewServer(redisOpt, 20)

	srv.Handle(asynqqueue.TypeProcessJob, func(ctx context.Context, task *asynq.Task) error {
		p, err := asynqqueue.ParseJobPayload(task)
		if err != nil {
			return err
		}
		return app.Engine.ProcessJob(ctx, p.EnvID, p.JobID)
	})

	srv.Handle(asynqqueue.TypeDigestFlush, func(ctx context.Context, task *asynq.Task) error {
		p, err := asynqqueue.ParseDigestPayload(task)
		if err != nil {
			return err
		}
		return app.Engine.FlushDigest(ctx, p.EnvID, p.BucketID, p.JobID)
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

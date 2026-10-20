package bootstrap

import (
	"context"
	"log/slog"

	"github.com/SolaTyolo/herald/internal/config"
	"github.com/SolaTyolo/herald/internal/delivery"
	"github.com/SolaTyolo/herald/internal/delivery/resultqueue"
	asynqqueue "github.com/SolaTyolo/herald/internal/queue/asynq"
	"github.com/SolaTyolo/herald/internal/realtime/bridge"
	"github.com/SolaTyolo/herald/internal/repository"
	"github.com/SolaTyolo/herald/internal/routing"
	"github.com/SolaTyolo/herald/internal/service"
	"github.com/SolaTyolo/herald/internal/telemetry"
	"github.com/SolaTyolo/herald/internal/transport/http/handler"
	"github.com/SolaTyolo/herald/internal/workflow"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

type localTasks struct {
	engine  *workflow.Engine
	trigger *service.TriggerService
}

func (t *localTasks) ProcessJob(ctx context.Context, jobID string) error {
	return t.engine.ProcessJob(ctx, jobID)
}

func (t *localTasks) FlushDigest(ctx context.Context, bucketID, jobID string) error {
	return t.engine.FlushDigest(ctx, bucketID, jobID)
}

func (t *localTasks) ProcessTopicFanout(ctx context.Context, p asynqqueue.TopicFanoutPayload) error {
	return t.trigger.ProcessTopicFanout(ctx, p)
}

type Role int

const (
	RoleAPI Role = iota
	RoleWorker
)

type App struct {
	Config    *config.Config
	Store     repository.Store
	Redis     *redis.Client
	Queue     asynqqueue.Enqueuer
	BridgeSub bridge.Subscriber
	Routes    *routing.Table
	Engine    *workflow.Engine
	Trigger   *service.TriggerService
	Handler   *handler.API
	Shutdown  func(context.Context) error
	stopWatch func()
}

func New(ctx context.Context, role Role) (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	if role == RoleAPI {
		cfg.OTelServiceName = "herald-api"
	} else {
		cfg.OTelServiceName = "herald-worker"
	}

	shutdown, err := telemetry.Setup(ctx, cfg)
	if err != nil {
		return nil, err
	}

	store, err := repository.Open(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if role == RoleAPI {
		if err := store.RunMigrations(ctx); err != nil {
			store.Close()
			return nil, err
		}
	}

	var redisClient *redis.Client
	var queueClient asynqqueue.Enqueuer
	if cfg.UsesAsynq() {
		redisClient = redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
		queueClient = asynqqueue.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisAddr})
	} else {
		queueClient = asynqqueue.NewLocal()
	}

	var bridgeSub bridge.Subscriber
	eventPub, err := bridge.OpenPublisher(cfg, redisClient)
	if err != nil {
		store.Close()
		return nil, err
	}
	if role == RoleAPI {
		webhookDeliverer := delivery.NewInAppWebhookDeliverer(store)
		bridgeSub, err = bridge.OpenSubscriber(cfg, redisClient)
		if err != nil {
			store.Close()
			return nil, err
		}
		bridgeSub.Run(ctx, webhookDeliverer.Handle)
	}

	routes, err := routing.Open(cfg.ProvidersConfig)
	if err != nil {
		store.Close()
		return nil, err
	}
	stopWatch, err := routing.Watch(ctx, routes)
	if err != nil {
		slog.Warn("providers config watch disabled", "path", cfg.ProvidersConfig, "err", err)
		stopWatch = func() {}
	}
	routes.StartHealthChecks(ctx)
	slog.Info("providers routing loaded", "path", cfg.ProvidersConfig, "providers", len(routes.ListProviders()))

	var results *resultqueue.Queue
	asyncResults := cfg.DeliveryResultMode == "async"
	if asyncResults {
		results = resultqueue.New(redisClient)
	}
	deliverySvc := delivery.New(store, routes, eventPub, delivery.Options{
		AsyncResults: asyncResults,
		Results:      results,
		SMTP: delivery.SMTPConfig{
			Host:     cfg.SMTPHost,
			Port:     cfg.SMTPPort,
			From:     cfg.SMTPFrom,
			Username: cfg.SMTPUsername,
			Password: cfg.SMTPPassword,
		},
	})
	engine := workflow.NewEngine(store, queueClient, deliverySvc, redisClient)
	triggerSvc := service.NewTriggerService(store, queueClient, cfg.MaxRecipients)
	if local, ok := queueClient.(*asynqqueue.Local); ok {
		local.Bind(&localTasks{engine: engine, trigger: triggerSvc})
	}

	app := &App{
		Config: cfg, Store: store, Redis: redisClient, Queue: queueClient,
		BridgeSub: bridgeSub, Routes: routes, Engine: engine, Trigger: triggerSvc,
		Shutdown: shutdown, stopWatch: stopWatch,
	}

	if role == RoleAPI {
		subscriberSvc := service.NewSubscriberService(store)
		workflowSvc := service.NewWorkflowService(store)
		topicSvc := service.NewTopicService(store)
		notificationSvc := service.NewNotificationQueryService(store)
		messageSvc := service.NewMessageService(store)
		templateSvc := service.NewTemplateService(store)

		app.Handler = handler.NewAPI(
			cfg,
			triggerSvc, subscriberSvc, workflowSvc, topicSvc,
			notificationSvc, messageSvc, templateSvc,
			store, redisClient, routes,
		)
	}

	return app, nil
}

func (a *App) Close() {
	if a.stopWatch != nil {
		a.stopWatch()
	}
	if a.BridgeSub != nil {
		a.BridgeSub.Close()
	}
	if a.Queue != nil {
		_ = a.Queue.Close()
	}
	_ = a.Store.Close()
	if a.Redis != nil {
		_ = a.Redis.Close()
	}
	if a.Shutdown != nil {
		_ = a.Shutdown(context.Background())
	}
}

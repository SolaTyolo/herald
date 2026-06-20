package bootstrap

import (
	"context"
	"log/slog"

	"github.com/SolaTyolo/herald/internal/config"
	"github.com/SolaTyolo/herald/internal/crypto"
	"github.com/SolaTyolo/herald/internal/delivery"
	wasmruntime "github.com/SolaTyolo/herald/internal/platform/plugin/wasm"
	asynqqueue "github.com/SolaTyolo/herald/internal/queue/asynq"
	"github.com/SolaTyolo/herald/internal/realtime/bridge"
	"github.com/SolaTyolo/herald/internal/repository"
	"github.com/SolaTyolo/herald/internal/service"
	"github.com/SolaTyolo/herald/internal/telemetry"
	"github.com/SolaTyolo/herald/internal/transport/http/handler"
	"github.com/SolaTyolo/herald/internal/workflow"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

type Role int

const (
	RoleAPI Role = iota
	RoleWorker
)

type App struct {
	Config      *config.Config
	Store       repository.Store
	Redis       *redis.Client
	Queue       *asynqqueue.Client
	BridgeSub   bridge.Subscriber
	WASM        *wasmruntime.Runtime
	Engine      *workflow.Engine
	Handler     *handler.API
	Encryptor   *crypto.Encryptor
	Shutdown    func(context.Context) error
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
	env, apiKey, err := store.EnsureDefaultTenant(ctx)
	if err != nil {
		store.Close()
		return nil, err
	}
	if apiKey != "" {
		slog.Info("default api key created", "prefix", apiKey[:10], "env", env.Slug, "hint", "full key printed once at creation — store it securely")
		if cfg.DevMode {
			slog.Warn("DEV_MODE: default API key", "key", apiKey)
		}
	}

	enc, err := crypto.NewEncryptor(cfg.EncryptionKey)
	if err != nil {
		store.Close()
		return nil, err
	}

	redisOpt := asynq.RedisClientOpt{Addr: cfg.RedisAddr}
	queueClient := asynqqueue.NewClient(redisOpt)
	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})

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

	wasmRT := wasmruntime.NewRuntime(cfg.WASMPluginDir)
	if err := wasmRT.LoadFromDir(); err != nil {
		store.Close()
		return nil, err
	}
	if len(wasmRT.ListManifests()) == 0 {
		slog.Warn("no wasm providers loaded", "dir", cfg.WASMPluginDir,
			"hint", "place provider.json + provider.wasm under WASM_PLUGIN_DIR")
	}

	deliverySvc := delivery.New(store, enc, wasmRT, eventPub)
	engine := workflow.NewEngine(store, queueClient, deliverySvc, redisClient)

	app := &App{
		Config: cfg, Store: store, Redis: redisClient, Queue: queueClient,
		BridgeSub: bridgeSub, WASM: wasmRT, Engine: engine, Encryptor: enc, Shutdown: shutdown,
	}

	if role == RoleAPI {
		triggerSvc := service.NewTriggerService(store, queueClient, cfg.MaxRecipients)
		subscriberSvc := service.NewSubscriberService(store)
		workflowSvc := service.NewWorkflowService(store)
		topicSvc := service.NewTopicService(store)
		integrationSvc := service.NewIntegrationService(store, enc, wasmRT)
		notificationSvc := service.NewNotificationQueryService(store)
		messageSvc := service.NewMessageService(store)

		app.Handler = handler.NewAPI(
			cfg,
			triggerSvc, subscriberSvc, workflowSvc, topicSvc,
			integrationSvc, notificationSvc, messageSvc,
			store, redisClient, wasmRT.ListManifests,
		)
	}

	return app, nil
}

func (a *App) Close() {
	if a.BridgeSub != nil {
		a.BridgeSub.Close()
	}
	a.Queue.Close()
	_ = a.Store.Close()
	_ = a.Redis.Close()
	if a.WASM != nil {
		_ = a.WASM.Close(context.Background())
	}
	if a.Shutdown != nil {
		_ = a.Shutdown(context.Background())
	}
}

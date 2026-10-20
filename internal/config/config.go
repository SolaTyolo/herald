package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type StoreType string

const (
	StoreTypeFile StoreType = "file"
	StoreTypeDB   StoreType = "db"
)

type DBDriver string

const (
	DBDriverPostgres DBDriver = "postgres"
	DBDriverMySQL    DBDriver = "mysql"
	DBDriverSQLite   DBDriver = "sqlite"
)

const (
	QueueModeLocal = "local"
	QueueModeAsynq = "asynq"
)

type Config struct {
	HTTPAddr        string
	APIKey          string
	StoreType       StoreType
	DBDriver        DBDriver
	DatabaseURL     string
	StoreFilePath   string
	RedisAddr       string
	ProvidersConfig string
	// QueueMode is local (in-process, no Redis) or asynq (Redis job queue, bridge, throttle, result queue).
	QueueMode string
	// DeliveryResultMode: sync (HTTP body) or async (Redis result queue; asynq only).
	DeliveryResultMode string
	MaxRecipients      int
	DevMode            bool

	// Builtin SMTP (smtp_email). Optional until an email step runs.
	SMTPHost     string
	SMTPPort     string
	SMTPFrom     string
	SMTPUsername string
	SMTPPassword string

	// Telemetry
	LogLevel           string
	LogFormat          string
	OTelEnabled        bool
	OTelEndpoint       string
	OTelServiceName    string
	OTelMetricsEnabled bool
	OTelTracesEnabled  bool
}

func Load() (*Config, error) {
	storeType := StoreType(strings.ToLower(getEnv("STORE_TYPE", "db")))
	if storeType != StoreTypeFile && storeType != StoreTypeDB {
		return nil, fmt.Errorf("STORE_TYPE must be file or db")
	}

	driver := DBDriver(strings.ToLower(getEnv("DB_DRIVER", "postgres")))
	if storeType == StoreTypeDB {
		switch driver {
		case DBDriverPostgres, DBDriverMySQL, DBDriverSQLite:
		default:
			return nil, fmt.Errorf("DB_DRIVER must be postgres, mysql, or sqlite")
		}
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = defaultDatabaseURL(driver)
	}

	queueMode, err := resolveQueueMode()
	if err != nil {
		return nil, err
	}
	redisAddr := ""
	if queueMode == QueueModeAsynq {
		redisAddr = getEnv("REDIS_ADDR", "localhost:6379")
	}
	resultMode := strings.ToLower(os.Getenv("DELIVERY_RESULT_MODE"))
	if resultMode == "" {
		if queueMode == QueueModeLocal {
			resultMode = "sync"
		} else {
			resultMode = "async"
		}
	}

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("API_KEY is required")
	}

	cfg := &Config{
		HTTPAddr:           getEnv("HTTP_ADDR", ":8080"),
		APIKey:             apiKey,
		StoreType:          storeType,
		DBDriver:           driver,
		DatabaseURL:        dbURL,
		StoreFilePath:      getEnv("STORE_FILE_PATH", "./data"),
		RedisAddr:          redisAddr,
		ProvidersConfig:    getEnv("PROVIDERS_CONFIG", "./config/providers.yaml"),
		QueueMode:          queueMode,
		DeliveryResultMode: resultMode,
		MaxRecipients:      getEnvInt("MAX_RECIPIENTS", 100),
		DevMode:            getEnvBool("DEV_MODE", false),
		SMTPHost:           os.Getenv("SMTP_HOST"),
		SMTPPort:           getEnv("SMTP_PORT", "587"),
		SMTPFrom:           os.Getenv("SMTP_FROM"),
		SMTPUsername:       os.Getenv("SMTP_USERNAME"),
		SMTPPassword:       os.Getenv("SMTP_PASSWORD"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		LogFormat:          getEnv("LOG_FORMAT", "text"),
		OTelEnabled:        getEnvBool("OTEL_ENABLED", false),
		OTelEndpoint:       getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318"),
		OTelServiceName:    getEnv("OTEL_SERVICE_NAME", "herald"),
		OTelMetricsEnabled: getEnvBool("OTEL_METRICS_ENABLED", true),
		OTelTracesEnabled:  getEnvBool("OTEL_TRACES_ENABLED", true),
	}
	switch cfg.DeliveryResultMode {
	case "sync", "async":
	default:
		return nil, fmt.Errorf("DELIVERY_RESULT_MODE must be sync or async")
	}
	if cfg.QueueMode == QueueModeLocal && cfg.DeliveryResultMode == "async" {
		return nil, fmt.Errorf("DELIVERY_RESULT_MODE=async requires QUEUE_MODE=asynq")
	}
	return cfg, nil
}

func (c *Config) UsesAsynq() bool {
	return c != nil && c.QueueMode == QueueModeAsynq
}

func resolveQueueMode() (string, error) {
	mode := strings.ToLower(getEnv("QUEUE_MODE", ""))
	if mode == "" {
		legacy := strings.ToLower(envFallback("WORKER_API_PUBSUB", "INAPP_PUBSUB", ""))
		switch legacy {
		case "", "redis":
			mode = QueueModeAsynq
		case "local":
			mode = QueueModeLocal
		default:
			return "", fmt.Errorf("WORKER_API_PUBSUB %q is removed; set QUEUE_MODE=local or asynq", legacy)
		}
	}
	switch mode {
	case QueueModeLocal, QueueModeAsynq:
		return mode, nil
	default:
		return "", fmt.Errorf("QUEUE_MODE must be local or asynq")
	}
}

func envFallback(primary, legacy, fallback string) string {
	if v := os.Getenv(primary); v != "" {
		return v
	}
	if v := os.Getenv(legacy); v != "" {
		return v
	}
	return fallback
}

func defaultDatabaseURL(driver DBDriver) string {
	switch driver {
	case DBDriverMySQL:
		return "herald:herald@tcp(localhost:3306)/herald?charset=utf8mb4&parseTime=True&Loc=Local"
	case DBDriverSQLite:
		return "./data/herald.db"
	default:
		return "postgres://herald:herald@localhost:5432/herald?sslmode=disable"
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return fallback
}

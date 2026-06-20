package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const defaultEncryptionKey = "0123456789abcdef0123456789abcdef"

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

type Config struct {
	HTTPAddr      string
	StoreType     StoreType
	DBDriver      DBDriver
	DatabaseURL   string
	StoreFilePath string
	RedisAddr     string
	EncryptionKey string
	WASMPluginDir string
	// Worker ↔ API event bus (in_app message fan-out to API for webhook delivery)
	WorkerAPIPubSub             string
	WorkerAPIRabbitMQHTTPURL    string
	WorkerAPIRabbitMQUser       string
	WorkerAPIRabbitMQPass       string
	WorkerAPIRabbitMQVHost      string
	WorkerAPIRabbitMQExchange   string
	WorkerAPIRabbitMQRoutingKey string
	WorkerAPIRabbitMQStreamURL  string
	WorkerAPIKafkaHTTPURL       string
	WorkerAPIKafkaTopic         string
	MaxRecipients int
	DevMode       bool

	// Telemetry
	LogLevel            string
	LogFormat           string
	OTelEnabled         bool
	OTelEndpoint        string
	OTelServiceName     string
	OTelMetricsEnabled  bool
	OTelTracesEnabled   bool
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

	cfg := &Config{
		HTTPAddr:          getEnv("HTTP_ADDR", ":8080"),
		StoreType:         storeType,
		DBDriver:          driver,
		DatabaseURL:       dbURL,
		StoreFilePath:     getEnv("STORE_FILE_PATH", "./data"),
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
		EncryptionKey:     getEnv("ENCRYPTION_KEY", defaultEncryptionKey),
		WASMPluginDir:     getEnv("WASM_PLUGIN_DIR", "./plugins/wasm"),
		WorkerAPIPubSub:             envFallback("WORKER_API_PUBSUB", "INAPP_PUBSUB", "redis"),
		WorkerAPIRabbitMQHTTPURL:    envFallback("WORKER_API_RABBITMQ_HTTP_URL", "INAPP_RABBITMQ_HTTP_URL", ""),
		WorkerAPIRabbitMQUser:       envFallback("WORKER_API_RABBITMQ_USER", "INAPP_RABBITMQ_USER", "guest"),
		WorkerAPIRabbitMQPass:       envFallback("WORKER_API_RABBITMQ_PASS", "INAPP_RABBITMQ_PASS", "guest"),
		WorkerAPIRabbitMQVHost:      envFallback("WORKER_API_RABBITMQ_VHOST", "INAPP_RABBITMQ_VHOST", "/"),
		WorkerAPIRabbitMQExchange:   envFallback("WORKER_API_RABBITMQ_EXCHANGE", "INAPP_RABBITMQ_EXCHANGE", "herald.worker-api"),
		WorkerAPIRabbitMQRoutingKey: envFallback("WORKER_API_RABBITMQ_ROUTING_KEY", "INAPP_RABBITMQ_ROUTING_KEY", "events"),
		WorkerAPIRabbitMQStreamURL:  envFallback("WORKER_API_RABBITMQ_STREAM_URL", "INAPP_RABBITMQ_STREAM_URL", ""),
		WorkerAPIKafkaHTTPURL:       envFallback("WORKER_API_KAFKA_HTTP_URL", "INAPP_KAFKA_HTTP_URL", ""),
		WorkerAPIKafkaTopic:         envFallback("WORKER_API_KAFKA_TOPIC", "INAPP_KAFKA_TOPIC", "herald.worker-api.events"),
		MaxRecipients:     getEnvInt("MAX_RECIPIENTS", 100),
		DevMode:           getEnvBool("DEV_MODE", false),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		LogFormat:         getEnv("LOG_FORMAT", "text"),
		OTelEnabled:       getEnvBool("OTEL_ENABLED", false),
		OTelEndpoint:      getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318"),
		OTelServiceName:   getEnv("OTEL_SERVICE_NAME", "herald"),
		OTelMetricsEnabled: getEnvBool("OTEL_METRICS_ENABLED", true),
		OTelTracesEnabled:  getEnvBool("OTEL_TRACES_ENABLED", true),
	}
	if len(cfg.EncryptionKey) != 32 {
		return nil, fmt.Errorf("ENCRYPTION_KEY must be 32 bytes")
	}
	if !cfg.DevMode && cfg.EncryptionKey == defaultEncryptionKey {
		return nil, fmt.Errorf("refusing default ENCRYPTION_KEY outside DEV_MODE; set ENCRYPTION_KEY or DEV_MODE=true")
	}
	switch strings.ToLower(cfg.WorkerAPIPubSub) {
	case "redis", "local", "rabbitmq-http", "kafka-http":
	default:
		return nil, fmt.Errorf("WORKER_API_PUBSUB must be redis, local, rabbitmq-http, or kafka-http")
	}
	if err := validateWorkerAPIPubSub(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func validateWorkerAPIPubSub(cfg *Config) error {
	switch strings.ToLower(cfg.WorkerAPIPubSub) {
	case "rabbitmq-http":
		if cfg.WorkerAPIRabbitMQHTTPURL == "" {
			return fmt.Errorf("WORKER_API_RABBITMQ_HTTP_URL required when WORKER_API_PUBSUB=rabbitmq-http")
		}
		if cfg.WorkerAPIRabbitMQStreamURL == "" {
			return fmt.Errorf("WORKER_API_RABBITMQ_STREAM_URL required when WORKER_API_PUBSUB=rabbitmq-http")
		}
	case "kafka-http":
		if cfg.WorkerAPIKafkaHTTPURL == "" {
			return fmt.Errorf("WORKER_API_KAFKA_HTTP_URL required when WORKER_API_PUBSUB=kafka-http")
		}
	}
	return nil
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

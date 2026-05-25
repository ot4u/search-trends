package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type HTTP struct {
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}
type NATS struct {
	URL           string
	Stream        string
	Subject       string
	Durable       string
	FetchBatch    int
	FetchTimeout  time.Duration
	AckWait       time.Duration
	MaxAckPending int
	AutoProvision bool
}
type Config struct {
	HTTP                        HTTP
	NATS                        NATS
	LogLevel                    string
	ShutdownTimeout             time.Duration
	WindowSeconds               int
	QueueSize                   int
	MaxUniqueQueries            int
	MaxFutureSkew               time.Duration
	AntiFraudThresholdPerSecond uint64
	SnapshotLimits              []int
	Stoplist                    []string
}

func Load() (Config, error) {
	cfg := Config{
		HTTP: HTTP{
			Addr:         envString("HTTP_ADDR", ":8080"),
			ReadTimeout:  envDuration("HTTP_READ_TIMEOUT", 5*time.Second),
			WriteTimeout: envDuration("HTTP_WRITE_TIMEOUT", 5*time.Second),
			IdleTimeout:  envDuration("HTTP_IDLE_TIMEOUT", 30*time.Second),
		},
		NATS: NATS{
			URL:           envString("NATS_URL", "nats://127.0.0.1:4222"),
			Stream:        envString("NATS_STREAM", "search-events"),
			Subject:       envString("NATS_SUBJECT", "search.events"),
			Durable:       envString("NATS_DURABLE", "search-trends"),
			FetchBatch:    envInt("NATS_FETCH_BATCH", 256),
			FetchTimeout:  envDuration("NATS_FETCH_TIMEOUT", time.Second),
			AckWait:       envDuration("NATS_ACK_WAIT", 30*time.Second),
			MaxAckPending: envInt("NATS_MAX_ACK_PENDING", 20_000),
			AutoProvision: envBool("NATS_AUTO_PROVISION", true),
		},
		LogLevel:                    envString("LOG_LEVEL", "info"),
		ShutdownTimeout:             envDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		WindowSeconds:               envInt("WINDOW_SECONDS", 300),
		QueueSize:                   envInt("QUEUE_SIZE", 65_536),
		MaxUniqueQueries:            envInt("MAX_UNIQUE_QUERIES", 200_000),
		MaxFutureSkew:               envDuration("MAX_FUTURE_SKEW", 2*time.Second),
		AntiFraudThresholdPerSecond: envUint64("ANTIFRAUD_THRESHOLD_PER_SECOND", 20),
		SnapshotLimits:              envIntSlice("SNAPSHOT_LIMITS", []int{5, 10, 25, 50, 100}),
		Stoplist:                    envStringSlice("STOPLIST_WORDS", nil),
	}

	if cfg.WindowSeconds <= 0 {
		return Config{}, fmt.Errorf("WINDOW_SECONDS must be positive")
	}
	if cfg.QueueSize <= 0 {
		return Config{}, fmt.Errorf("QUEUE_SIZE must be positive")
	}
	if cfg.MaxUniqueQueries < 0 {
		return Config{}, fmt.Errorf("MAX_UNIQUE_QUERIES must be >= 0")
	}
	if len(cfg.SnapshotLimits) == 0 {
		return Config{}, fmt.Errorf("SNAPSHOT_LIMITS must not be empty")
	}

	return cfg, nil
}

func envString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envUint64(key string, fallback uint64) uint64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envIntSlice(key string, fallback []int) []int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parts := strings.Split(value, ",")
	items := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		parsed, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		items = append(items, parsed)
	}

	if len(items) == 0 {
		return fallback
	}
	return items
}

func envStringSlice(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		items = append(items, part)
	}

	if len(items) == 0 {
		return fallback
	}
	return items
}

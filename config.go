package edge

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ManifestURL     string
	DeviceToken     string
	PollInterval    time.Duration
	DataDir         string
	StateFile       string
	HTTPTimeout     time.Duration
	WorkerCount     int
	MaxAttempts     int
	DownloadTimeout time.Duration
}

const (
	defaultManifestURL     = "http://localhost:8080/v2/manifest"
	defaultPollInterval    = 30 * time.Second
	defaultHTTPTimeout     = 10 * time.Second
	defaultDataDir         = "/tmp/winnow"
	defaultWorkerCount     = 4
	defaultMaxAttempts     = 3
	defaultDownloadTimeout = 15 * time.Second
)

func FromEnv() Config {
	dataDir := getEnv("DATA_DIR", defaultDataDir)
	cfg := Config{
		ManifestURL:     getEnv("MANIFEST_URL", defaultManifestURL),
		DeviceToken:     os.Getenv("DEVICE_TOKEN"),
		PollInterval:    parseDuration(getEnv("POLL_INTERVAL", defaultPollInterval.String()), defaultPollInterval),
		DataDir:         dataDir,
		StateFile:       getEnv("STATE_FILE", dataDir+"/state.json"),
		HTTPTimeout:     parseDuration(getEnv("HTTP_TIMEOUT", defaultHTTPTimeout.String()), defaultHTTPTimeout),
		WorkerCount:     parseInt(getEnv("WORKER_COUNT", fmt.Sprintf("%d", defaultWorkerCount)), defaultWorkerCount),
		MaxAttempts:     parseInt(getEnv("MAX_ATTEMPTS", fmt.Sprintf("%d", defaultMaxAttempts)), defaultMaxAttempts),
		DownloadTimeout: parseDuration(getEnv("DOWNLOAD_TIMEOUT", defaultDownloadTimeout.String()), defaultDownloadTimeout),
	}

	slog.Info("loaded config",
		"manifest_url", cfg.ManifestURL,
		"poll_interval", cfg.PollInterval,
		"data_dir", cfg.DataDir,
		"state_file", cfg.StateFile,
		"http_timeout", cfg.HTTPTimeout,
		"worker_count", cfg.WorkerCount,
		"max_attempts", cfg.MaxAttempts,
		"download_timeout", cfg.DownloadTimeout,
	)

	return cfg
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func parseDuration(raw string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return d
}

func parseInt(raw string, fallback int) int {
	i, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return i
}

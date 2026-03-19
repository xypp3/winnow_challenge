package edge

import (
	"log/slog"
	"os"
	"time"
)

type Config struct {
	ManifestURL  string
	DeviceToken  string
	PollInterval time.Duration
	DataDir      string
	StateFile    string
	HTTPTimeout  time.Duration
}

const (
	defaultManifestURL  = "http://localhost:8080/v2/manifest"
	defaultPollInterval = 30 * time.Second
	defaultHTTPTimeout  = 10 * time.Second
	defaultDataDir      = "/tmp/winnow"
)

func FromEnv() Config {
	dataDir := getEnv("DATA_DIR", defaultDataDir)
	cfg := Config{
		ManifestURL:  getEnv("MANIFEST_URL", defaultManifestURL),
		DeviceToken:  os.Getenv("DEVICE_TOKEN"),
		PollInterval: parseDuration(getEnv("POLL_INTERVAL", defaultPollInterval.String()), defaultPollInterval),
		DataDir:      dataDir,
		StateFile:    getEnv("STATE_FILE", dataDir+"/state.json"),
		HTTPTimeout:  parseDuration(getEnv("HTTP_TIMEOUT", defaultHTTPTimeout.String()), defaultHTTPTimeout),
	}

	slog.Info("loaded config",
		"manifest_url", cfg.ManifestURL,
		"poll_interval", cfg.PollInterval,
		"data_dir", cfg.DataDir,
		"state_file", cfg.StateFile,
		"http_timeout", cfg.HTTPTimeout,
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

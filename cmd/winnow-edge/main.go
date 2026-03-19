package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	edge "winnow_edge_service"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := edge.FromEnv()
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		slog.Error("failed to create data dir", "error", err)
		os.Exit(1)
	}

	stateStore, err := edge.NewStore(cfg.StateFile)
	if err != nil {
		slog.Error("failed to load state", "error", err)
		os.Exit(1)
	}

	httpClient := &http.Client{Timeout: cfg.HTTPTimeout}
	manifestClient := edge.NewClient(cfg.ManifestURL, cfg.DeviceToken, httpClient)
	pub := edge.NewStdoutPublisher()
	svc := edge.New(cfg, manifestClient, stateStore, pub, httpClient)

	if err := svc.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("service stopped with error", "error", err)
		os.Exit(1)
	}
}

// havoc-recorder consumes experiment results from Kafka, persists every
// result to Postgres as an immutable row, releases the per-service
// active-experiment lock the control plane took out, and emits
// structured JSON logs for Filebeat → ELK.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/hashir-Zahoor-kh/Havoc/internal/config"
	hkafka "github.com/hashir-Zahoor-kh/Havoc/internal/kafka"
	"github.com/hashir-Zahoor-kh/Havoc/internal/postgres"
	"github.com/hashir-Zahoor-kh/Havoc/internal/recorder"
	hredis "github.com/hashir-Zahoor-kh/Havoc/internal/redis"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "havoc-recorder:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.LoadRecorder()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})).
		With("component", "havoc-recorder")

	store, err := postgres.New(ctx, cfg.PostgresDSN)
	if err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	redis := hredis.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	defer redis.Close()

	consumer := hkafka.NewConsumer(cfg.KafkaBrokers, hkafka.TopicResults, cfg.KafkaGroupID)
	defer consumer.Close()

	rec := &recorder.Recorder{
		Experiments: store,
		Results:     store,
		Statuses:    store,
		Locks:       redis,
		Logger:      logger,
	}

	logger.Info("starting",
		"brokers", cfg.KafkaBrokers,
		"topic", hkafka.TopicResults,
		"group", cfg.KafkaGroupID,
	)

	// Errors from Process bubble out so the message offset stays
	// uncommitted and the orchestrator (Kubernetes) restarts the pod —
	// the result insert is idempotent so replays are safe. A real
	// deployment would split this into a retry-with-backoff loop and a
	// dead-letter queue for poison messages, but for a single-cluster
	// portfolio project crash-and-restart is good enough.
	err = consumer.Consume(ctx, func(ctx context.Context, _, value []byte) error {
		if err := rec.Process(ctx, value); err != nil {
			logger.Error("process failed", "err", err)
			return err
		}
		return nil
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	logger.Info("shut down")
	return nil
}

// havoc-agent runs as a Kubernetes DaemonSet — one pod per node — and
// is the binary that actually injects faults. It consumes domain.Command
// envelopes from the havoc.commands topic, intersects each command's
// pre-picked TargetPods with what is scheduled on its own node, runs
// the requested chaos action, and publishes a domain.ExperimentResult.
//
// The Kafka consumer group is keyed by node name so every agent gets
// its own independent partition assignment — i.e. every command lands
// on every agent. Self-filtering happens inside the runner.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/hashir-Zahoor-kh/Havoc/internal/agent"
	"github.com/hashir-Zahoor-kh/Havoc/internal/chaos"
	"github.com/hashir-Zahoor-kh/Havoc/internal/config"
	"github.com/hashir-Zahoor-kh/Havoc/internal/health"
	"github.com/hashir-Zahoor-kh/Havoc/internal/k8s"
	hkafka "github.com/hashir-Zahoor-kh/Havoc/internal/kafka"
	hredis "github.com/hashir-Zahoor-kh/Havoc/internal/redis"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "havoc-agent:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.LoadAgent()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})).
		With("component", "havoc-agent", "node", cfg.NodeName)

	// Health server starts immediately so the kubelet sees /healthz
	// 200 even while the heavier dependencies are still being dialed.
	// /readyz stays 503 until MarkReady is called below.
	healthSrv := health.New(cfg.HealthAddr)
	go func() {
		if err := healthSrv.Start(ctx); err != nil {
			logger.Error("health server stopped", "err", err)
		}
	}()

	kc, err := k8s.New(k8s.Config{InCluster: cfg.InCluster, KubeconfigPath: cfg.KubeconfigPath})
	if err != nil {
		return fmt.Errorf("k8s: %w", err)
	}

	redis := hredis.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	defer redis.Close()

	results := hkafka.NewProducer(cfg.KafkaBrokers, hkafka.TopicResults)
	defer results.Close()

	// Per-agent consumer group: each agent must see every command so it
	// can decide for itself whether to act. Sharing one group across
	// agents would let Kafka load-balance the messages, which is the
	// opposite of what we want here.
	groupID := cfg.KafkaGroupID + "-" + cfg.NodeName
	commands := hkafka.NewConsumer(cfg.KafkaBrokers, hkafka.TopicCommands, groupID)
	defer commands.Close()

	registry := chaos.NewRegistry(
		&chaos.PodKiller{K8s: kc},
		&chaos.CPUPressure{K8s: kc},
		&chaos.NetworkLatency{K8s: kc},
	)
	runner := agent.New(cfg.NodeName, kc, redis, results, registry, logger)

	logger.Info("starting",
		"brokers", cfg.KafkaBrokers,
		"topic", hkafka.TopicCommands,
		"group", groupID,
		"health_addr", cfg.HealthAddr,
	)

	// All dependencies dialed successfully — flip /readyz to 200 so
	// the kubelet (and any service mesh) starts treating this pod as
	// part of the rotation.
	healthSrv.MarkReady()

	// Handle returns nil for poison messages so they're committed and
	// skipped. Real I/O failures (Kafka itself going away, ctx
	// cancellation) bubble out and let Kubernetes restart the pod.
	err = commands.Consume(ctx, func(ctx context.Context, _, value []byte) error {
		return runner.Handle(ctx, value)
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	logger.Info("shut down")
	return nil
}

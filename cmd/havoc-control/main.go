// havoc-control is the control-plane HTTP API and the operator CLI for
// the Havoc chaos engineering platform.
//
//	havoc-control serve                 — run the API server
//	havoc-control schedule ...          — submit an experiment
//	havoc-control list                  — show recent experiments
//	havoc-control stop --id <id>        — abort a single experiment
//	havoc-control stop-all              — engage the global kill switch
//	havoc-control resume                — clear the global kill switch
//	havoc-control blackout add|list|rm  — manage blackout windows
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if len(os.Args) < 2 {
		usage(os.Stderr)
		os.Exit(2)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "serve":
		err = runServe(ctx, args)
	case "schedule":
		err = runSchedule(ctx, args)
	case "list":
		err = runList(ctx, args)
	case "stop":
		err = runStop(ctx, args)
	case "stop-all":
		err = runStopAll(ctx, args)
	case "resume":
		err = runResume(ctx, args)
	case "blackout":
		err = runBlackout(ctx, args)
	case "-h", "--help", "help":
		usage(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		usage(os.Stderr)
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage(w *os.File) {
	fmt.Fprintln(w, `usage: havoc-control <command> [flags]

Server:
  serve                        run the HTTP API server

Experiments:
  schedule --action <a> --target k=v --namespace <ns> --duration <s> [...]
  list                         show recent experiments
  stop --id <experiment-id>    abort a single experiment
  stop-all                     engage the global kill switch
  resume                       clear the global kill switch

Blackout windows:
  blackout add --name <n> --cron <expr> --duration <m>
  blackout list
  blackout rm <name>

Environment:
  HAVOC_API_URL                base URL of the control-plane API (CLI mode)
  HAVOC_HTTP_ADDR              listen address for serve (default :8080)
  HAVOC_KAFKA_BROKERS          comma-separated broker list
  HAVOC_POSTGRES_DSN           Postgres connection string
  HAVOC_REDIS_ADDR             Redis host:port
  HAVOC_BLAST_RADIUS_PCT       max % of matching pods an experiment may affect
  HAVOC_IN_CLUSTER             "true" to load Kubernetes credentials from the pod`)
}

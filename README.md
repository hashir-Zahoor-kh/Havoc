# Havoc

A controlled chaos engineering platform for Kubernetes. Havoc schedules experiments, kill a pod, inject latency, burn CPU, and executes them through lightweight agents running on every cluster node, behind hard safety guardrails that prevent an experiment from escaping its intended blast radius. Every experiment is recorded in an auditable, searchable history.

This is a portfolio project. It is production-shaped, not production-sized: a working end-to-end demo of a distributed command-and-control architecture.

## Architecture

Three Go binaries, one module, one repository.

```
  CLI ─▶  Control Plane ─▶  Kafka (havoc.commands)  ─▶  Agents (DaemonSet)
                │                                             │
                ▼                                             ▼
         Postgres / Redis                         Kafka (havoc.results)
                                                              │
                                                              ▼
                                                          Recorder
                                                              │
                                                              ▼
                                                 Postgres + Redis + ELK
```

### Components

| Binary | Role |
| --- | --- |
| `havoc-control` | HTTP API + CLI. Validates experiments, enforces guardrails, publishes commands. |
| `havoc-agent`   | DaemonSet. Consumes commands, executes chaos actions on its node, publishes results. |
| `havoc-recorder`| Kafka consumer. Writes results to Postgres, clears Redis locks, emits structured logs. |

## Chaos Actions

Three, deliberately.

- **Pod Kill** — deletes a randomly-selected pod matching a label selector.
- **Network Latency** — injects `tc`-based outbound latency inside the target pod's network namespace.
- **CPU Pressure** — runs a CPU-burn routine inside the target pod for a defined duration.

## Safety Guardrails

- **Blast radius limit.** Every experiment declares a label selector. The control plane rejects it if the experiment would affect more than N% of matching pods (default 25%).
- **Active-experiment lock.** A Redis key `havoc:active:{service}` blocks stacking two experiments on the same service.
- **Global kill switch.** A single Redis key `havoc:killswitch` that every agent checks before acting. `havoc stop-all` sets it.
- **Blackout windows.** Config rows in Postgres describing time ranges during which experiments are rejected (e.g. business hours, deployment freezes).

## Tech Stack

Go · Apache Kafka · PostgreSQL · Redis · ELK · Kubernetes (AKS + kind) · Terraform · Helm · Docker · GitLab CI/CD

## Repository Layout

```
havoc/
├── cmd/
│   ├── havoc-control/    # API server + CLI
│   ├── havoc-agent/      # DaemonSet binary
│   └── havoc-recorder/   # Kafka consumer → Postgres
├── internal/
│   ├── domain/           # experiment types + validation
│   ├── kafka/            # producer + consumer wrappers
│   ├── postgres/         # queries + migrations helper
│   ├── redis/            # kill switch + lock helpers
│   ├── chaos/            # the three action implementations
│   └── safety/           # guardrail logic
├── migrations/           # SQL migrations
├── deployments/
│   ├── docker/           # Dockerfiles
│   ├── helm/             # Helm charts
│   └── terraform/        # Azure resources
├── dashboards/kibana/    # Kibana saved searches
├── docker-compose.yml    # local stack
└── .gitlab-ci.yml
```

## Getting Started

Coming soon. Local development uses docker-compose for Kafka / Postgres / Redis / ELK and a `kind` cluster for the agents.

## Status

Phase 1 — scaffold and domain types — in progress. See the build plan in commits for progress through subsequent phases.

## License

MIT. See [LICENSE](LICENSE).

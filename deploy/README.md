# Local Havoc stack

This directory bundles everything needed to run Havoc end-to-end on a
laptop: a docker-compose stack for the infra Havoc depends on (Kafka,
Postgres, Redis, ELK), a kind cluster manifest for the Kubernetes side,
container build files for the three Havoc binaries, and the k8s
manifests that wire it all up.

The Makefile at the repo root drives every step. `make help` lists the
targets; the prose below is the why.

## Layout

```
deploy/
├── docker-compose.yml      # Kafka, ZK, Postgres, Redis, ES, Kibana
├── kind/cluster.yaml       # 1 control-plane + 2 worker kind cluster
├── Dockerfile              # Multi-stage, parameterized by --build-arg BIN=
├── bootstrap/              # Kafka topic creation
└── k8s/                    # Namespaces, RBAC, three workloads, Filebeat,
                            # demo target
```

## End-to-end loop

From a clean laptop:

```sh
make all-up    # compose up + bootstrap + kind + images + load + deploy
make demo      # schedule a pod_kill against the checkout demo workload
make logs-agent
```

That sequence:

1. Brings up Kafka/ZK/Postgres/Redis/ELK on a docker network named `havoc`.
2. Creates the two Kafka topics `havoc.commands` and `havoc.results`.
3. Creates a 3-node kind cluster and joins each node to the `havoc`
   docker network so cluster pods can reach `kafka:29092` etc. by name.
4. Builds and side-loads the three Havoc images into kind.
5. Applies namespaces, RBAC, the control Deployment, the recorder
   Deployment, the agent DaemonSet, and a six-replica `checkout` demo
   workload with `iproute2` (so `tc` works) and `NET_ADMIN` (so `tc`
   has permission to add a netem qdisc).
6. Schedules a pod_kill experiment via the CLI talking to
   `localhost:8080` (kind's extraPortMappings publishes the control
   plane's NodePort there).

`make all-down` reverses all of it. `make nuke` additionally drops the
docker volumes if you want a truly fresh slate.

## Why a single docker network for compose **and** kind

The agent container running inside a kind worker needs to reach the
Kafka broker that compose started on the host. Three options were on
the table:

- **Run Kafka inside kind too** — closer to a "real" cluster but
  doubles the resource cost and makes the local feedback loop slower.
- **Use `host.docker.internal`** — works on Docker Desktop but is
  fiddly on Linux.
- **Connect kind nodes to the compose network** — every container, no
  matter which compose project owns it, resolves `kafka` to the same
  IP. Cheap and uniform.

The Makefile takes the third option: after `kind create cluster`, every
kind node is attached to the `havoc` network. From the agent's point
of view, `kafka:29092` resolves the same way it would in production.

## Why a single multi-stage Dockerfile

`deploy/Dockerfile` is parameterized by `--build-arg BIN=havoc-agent`.
Three near-identical Dockerfiles would have drifted within a week; one
file means the toolchain version, build flags, and base image stay
identical across the trio.

## Why two namespaces

`havoc` holds the control plane, agents, and recorder. `havoc-demo`
holds the workload to be attacked. Splitting them makes the
blast-radius selector unambiguous and prevents an experiment from
ever targeting the control plane itself.

## Kibana

Filebeat runs as a DaemonSet **inside** the kind cluster (not in the
compose stack) and ships kubelet container logs from the `havoc`
namespace into the host Elasticsearch over the shared `havoc` docker
network. The host-side variant was tried first and didn't work — the
Havoc binaries run as nested containers inside kind nodes, so their
stdout never reaches the laptop's `/var/lib/docker/containers`.
Tailing `/var/log/pods` on each kind node catches the logs at the
right level.

The slog JSON line is decoded into top-level fields (`component`,
`experiment_id`, `outcome`, `agent_node`, etc.) so a query like
`experiment_id:abc-123` returns every log line from every component
that touched that experiment. `make kibana` opens the UI.

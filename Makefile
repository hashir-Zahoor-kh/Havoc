# Local dev workflow for Havoc. The default target prints the available
# commands; the chain a fresh laptop runs is `up bootstrap kind images
# load deploy demo`.

SHELL := /usr/bin/env bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := help

COMPOSE := docker compose -f deploy/docker-compose.yml
KIND_NAME := havoc
KUBECTL := kubectl

# Bumping IMAGE_TAG forces a rebuild + reload — useful when iterating
# on the binaries.
IMAGE_TAG ?= dev

BINARIES := havoc-control havoc-agent havoc-recorder

.PHONY: help
help: ## Show available targets
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'

# ---------- Go build / test ----------

.PHONY: build
build: ## Build all three binaries to ./bin
	@mkdir -p bin
	@for b in $(BINARIES); do \
		echo "build $$b"; \
		go build -trimpath -o bin/$$b ./cmd/$$b; \
	done

.PHONY: test
test: ## Run unit tests
	go test ./...

.PHONY: vet
vet: ## go vet
	go vet ./...

# ---------- Local infra ----------

.PHONY: up
up: ## Start Kafka, Postgres, Redis, ELK via docker compose
	$(COMPOSE) up -d
	@echo "waiting for kafka..."
	@until $(COMPOSE) exec -T kafka kafka-topics --bootstrap-server localhost:9092 --list >/dev/null 2>&1; do sleep 2; done
	@echo "ready"

.PHONY: down
down: ## Stop the docker-compose stack (keeps volumes)
	$(COMPOSE) down

.PHONY: nuke
nuke: ## Stop the stack AND drop all volumes
	$(COMPOSE) down -v

.PHONY: bootstrap
bootstrap: ## Create the havoc.commands and havoc.results Kafka topics
	$(COMPOSE) exec -T kafka /bin/sh < deploy/bootstrap/create-topics.sh

# ---------- kind cluster ----------

.PHONY: kind
kind: ## Create the kind cluster and join it to the havoc docker network
	kind create cluster --name $(KIND_NAME) --config deploy/kind/cluster.yaml
	# kind builds its own bridge; connecting each node to the havoc
	# network lets pods inside kind reach `kafka:29092` etc. by service
	# name. Idempotent — `network connect` errors on re-run, swallow it.
	@for node in $$(kind get nodes --name $(KIND_NAME)); do \
		docker network connect havoc $$node 2>/dev/null || true; \
	done

.PHONY: kind-down
kind-down: ## Tear down the kind cluster
	kind delete cluster --name $(KIND_NAME)

# ---------- Container images ----------

.PHONY: images
images: ## Build the three Havoc container images
	@for b in $(BINARIES); do \
		echo "build image $$b:$(IMAGE_TAG)"; \
		docker build -f deploy/Dockerfile --build-arg BIN=$$b -t havoc/$$b:$(IMAGE_TAG) .; \
	done

.PHONY: load
load: ## Load the three images into the kind cluster
	@for b in $(BINARIES); do \
		kind load docker-image havoc/$$b:$(IMAGE_TAG) --name $(KIND_NAME); \
	done

# ---------- Deploy + demo ----------

.PHONY: deploy
deploy: ## Apply all k8s manifests
	$(KUBECTL) apply -f deploy/k8s/namespace.yaml
	$(KUBECTL) apply -f deploy/k8s/rbac.yaml
	$(KUBECTL) apply -f deploy/k8s/havoc-control.yaml
	$(KUBECTL) apply -f deploy/k8s/havoc-recorder.yaml
	$(KUBECTL) apply -f deploy/k8s/havoc-agent.yaml
	$(KUBECTL) apply -f deploy/k8s/filebeat.yaml
	$(KUBECTL) apply -f deploy/k8s/demo-workload.yaml
	$(KUBECTL) -n havoc rollout status deploy/havoc-control --timeout=120s
	$(KUBECTL) -n havoc rollout status deploy/havoc-recorder --timeout=120s
	$(KUBECTL) -n havoc rollout status ds/havoc-agent --timeout=120s
	$(KUBECTL) -n havoc rollout status ds/filebeat --timeout=120s
	$(KUBECTL) -n havoc-demo rollout status deploy/checkout --timeout=180s

.PHONY: undeploy
undeploy: ## Delete every Havoc resource
	$(KUBECTL) delete -f deploy/k8s/demo-workload.yaml --ignore-not-found
	$(KUBECTL) delete -f deploy/k8s/filebeat.yaml --ignore-not-found
	$(KUBECTL) delete -f deploy/k8s/havoc-agent.yaml --ignore-not-found
	$(KUBECTL) delete -f deploy/k8s/havoc-recorder.yaml --ignore-not-found
	$(KUBECTL) delete -f deploy/k8s/havoc-control.yaml --ignore-not-found
	$(KUBECTL) delete -f deploy/k8s/rbac.yaml --ignore-not-found
	$(KUBECTL) delete -f deploy/k8s/namespace.yaml --ignore-not-found

.PHONY: demo
demo: ## Schedule one pod_kill experiment against the checkout app
	go run ./cmd/havoc-control schedule \
		--action pod_kill \
		--namespace havoc-demo \
		--target app=checkout \
		--duration 10s

.PHONY: demo-cpu
demo-cpu: ## Schedule one cpu_pressure experiment
	go run ./cmd/havoc-control schedule \
		--action cpu_pressure \
		--namespace havoc-demo \
		--target app=checkout \
		--duration 30s \
		--cpu-percent 80

.PHONY: demo-latency
demo-latency: ## Schedule one network_latency experiment
	go run ./cmd/havoc-control schedule \
		--action network_latency \
		--namespace havoc-demo \
		--target app=checkout \
		--duration 30s \
		--latency 200ms

# ---------- Convenience ----------

.PHONY: logs-control
logs-control: ## Tail control-plane logs
	$(KUBECTL) -n havoc logs deploy/havoc-control -f

.PHONY: logs-agent
logs-agent: ## Tail agent logs (all pods)
	$(KUBECTL) -n havoc logs ds/havoc-agent -f --max-log-requests 6

.PHONY: logs-recorder
logs-recorder: ## Tail recorder logs
	$(KUBECTL) -n havoc logs deploy/havoc-recorder -f

.PHONY: psql
psql: ## Open psql against the local Postgres (uses the container's client so no host psql is needed)
	$(COMPOSE) exec postgres psql -U havoc -d havoc

.PHONY: kibana
kibana: ## Open Kibana in a browser
	open http://localhost:5601 || xdg-open http://localhost:5601

.PHONY: all-up
all-up: up bootstrap kind images load deploy ## End-to-end: bring up everything

.PHONY: all-down
all-down: undeploy kind-down down ## End-to-end: tear down everything

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

# AWS region for ECR + EKS commands. Kept here (not behind `aws
# configure`) so the Makefile is self-describing and CI can override
# without a profile flip.
AWS_REGION ?= us-east-1

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
all-up: up bootstrap kind images load deploy ## End-to-end via raw kubectl manifests: compose + kind + images + load + apply

.PHONY: all-down
all-down: undeploy kind-down down ## End-to-end teardown of the kubectl-apply path

# ---------- Helm path (Phase 6b) ----------
#
# `local-up` brings the local stack up to "images loaded" but does
# NOT apply any manifests — it leaves the cluster empty so `helm-up`
# can install the charts cleanly. The kubectl-apply path
# (`make all-up`) and the helm path are mutually exclusive on the
# same kind cluster — running both will create conflicting
# resources.

.PHONY: local-up
local-up: up bootstrap kind images load ## Helm path: compose + kind + images + load (no manifests applied)

.PHONY: helm-up
helm-up: ## Install both charts against the kind cluster
	helm upgrade --install havoc charts/havoc -n havoc --create-namespace --wait --timeout 3m
	helm upgrade --install havoc-demo charts/havoc-demo -n havoc-demo --create-namespace --wait --timeout 3m

.PHONY: helm-down
helm-down: ## Uninstall both charts (kind cluster + compose stay)
	-helm uninstall havoc-demo -n havoc-demo
	-helm uninstall havoc -n havoc

.PHONY: helm-template
helm-template: ## Render templates locally without contacting any cluster (dry-run lint)
	helm template havoc charts/havoc -n havoc > /tmp/havoc-rendered.yaml
	helm template havoc-demo charts/havoc-demo -n havoc-demo > /tmp/havoc-demo-rendered.yaml
	@echo "rendered to /tmp/havoc-rendered.yaml and /tmp/havoc-demo-rendered.yaml"

.PHONY: helm-lint
helm-lint: ## helm lint both charts
	helm lint charts/havoc
	helm lint charts/havoc-demo

# ---------- AWS (Terraform) ----------
#
# These targets run from terraform/ so the backend config and provider
# cache stay in one place. `aws-up` is intentionally NOT chained
# into `all-up` — provisioning real cloud resources should always be
# an explicit, deliberate command. Run `make aws-down` at the end of
# every dev session; the EKS control plane alone is $0.10/hr whether
# you're using it or not.

TF := terraform -chdir=terraform

.PHONY: aws-init
aws-init: ## terraform init (run once, or after changing providers)
	$(TF) init

.PHONY: aws-plan
aws-plan: ## terraform plan against dev.tfvars
	$(TF) plan -var-file=dev.tfvars -out=plan.tfplan

.PHONY: aws-up
aws-up: ## terraform apply (provisions the dev environment in AWS — costs money)
	$(TF) apply -var-file=dev.tfvars

.PHONY: aws-down
aws-down: ## terraform destroy (tears down the dev environment)
	$(TF) destroy -var-file=dev.tfvars -auto-approve

.PHONY: aws-output
aws-output: ## Print non-sensitive outputs (cluster name, ECR registry, RDS/Redis endpoints)
	$(TF) output

.PHONY: aws-secrets
aws-secrets: ## Create/update the havoc-secrets Secret in EKS from terraform outputs
	# Read everything from terraform output (sensitive values via -raw)
	# and pipe straight into kubectl create. --dry-run + apply makes
	# this idempotent: re-running rotates the values in-place.
	#
	# The RDS admin password is generated from `!#$%^&*()-_=+[]{}<>?`
	# (see random_password.rds.override_special in postgres.tf).
	# Several of those chars — `{`, `}`, `:`, `@`, `/`, `?` — are
	# meaningful in URL syntax: a literal `:` in a password makes
	# the rest of the string parse as a port, a `@` ends the userinfo
	# component early, and so on. pgx errors with `invalid port`.
	# URL-encoding the password via python3 urllib.parse.quote
	# (safe="") percent-encodes every non-alphanumeric byte and
	# guarantees the DSN parses regardless of which chars the
	# generator picked this rotation.
	@RDS_HOST=$$($(TF) output -raw rds_endpoint); \
	 RDS_PORT=$$($(TF) output -raw rds_port); \
	 RDS_DB=$$($(TF) output -raw rds_database); \
	 RDS_USER=$$($(TF) output -raw rds_admin_user); \
	 RDS_PASS=$$($(TF) output -raw rds_admin_password); \
	 RDS_PASS_ENC=$$(printf '%s' "$${RDS_PASS}" | python3 -c 'import urllib.parse, sys; print(urllib.parse.quote(sys.stdin.read(), safe=""), end="")'); \
	 REDIS_HOST=$$($(TF) output -raw redis_endpoint); \
	 REDIS_PORT=$$($(TF) output -raw redis_port); \
	 PG_DSN="postgres://$${RDS_USER}:$${RDS_PASS_ENC}@$${RDS_HOST}:$${RDS_PORT}/$${RDS_DB}?sslmode=require"; \
	 REDIS_ADDR="$${REDIS_HOST}:$${REDIS_PORT}"; \
	 $(KUBECTL) create namespace havoc --dry-run=client -o yaml | $(KUBECTL) apply -f -; \
	 $(KUBECTL) -n havoc create secret generic havoc-secrets \
	   --from-literal=postgresDSN="$${PG_DSN}" \
	   --from-literal=redisAddr="$${REDIS_ADDR}" \
	   --from-literal=redisPassword="" \
	   --dry-run=client -o yaml | $(KUBECTL) apply -f -

# ---------- ECR (build + push) ----------
#
# The laptop is M-series (arm64); EKS nodes are x86_64. A plain
# `docker build` would produce arm64 images that simply error with
# `exec format error` on EKS. `docker buildx build --platform
# linux/amd64` does the cross-compile via QEMU emulation that ships
# with Docker Desktop. It's slow (~2-3 min per binary) but correct.
#
# `--push` requires the buildx default builder, which on Docker
# Desktop 4.20+ is the `desktop-linux` instance and supports
# multi-platform pushes natively. If you ever see "ERROR: Multiple
# platforms feature is currently not supported for docker driver",
# run `docker buildx create --name havoc-builder --use --bootstrap`
# once and the next push will use a docker-container builder that
# does support it.

.PHONY: ecr-login
ecr-login: ## Authenticate the local docker daemon to ECR (12h token)
	@REGISTRY=$$($(TF) output -raw ecr_registry); \
	 aws ecr get-login-password --region $(AWS_REGION) | \
	   docker login --username AWS --password-stdin $$REGISTRY

.PHONY: ecr-push
ecr-push: ecr-login ## Build the three images for linux/amd64 and push to ECR
	@REGISTRY=$$($(TF) output -raw ecr_registry); \
	 for b in $(BINARIES); do \
	   echo "build+push $$b:$(IMAGE_TAG) for linux/amd64"; \
	   docker buildx build \
	     --platform linux/amd64 \
	     --build-arg BIN=$$b \
	     -f deploy/Dockerfile \
	     -t $$REGISTRY/$$b:$(IMAGE_TAG) \
	     --push \
	     . ; \
	 done

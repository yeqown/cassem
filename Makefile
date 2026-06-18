.DEFAULT_GOAL := help

CONTAINER_TOOL ?= podman
IMAGE_TAG ?= latest

LDFLAGS := -s \
	-X main.Version=$(shell git describe --tags --abbrev=0 2>/dev/null || true) \
	-X main.BuildTime=$(shell TZ=UTC date -u '+%Y-%m-%dT%H:%M:%SZ') \
	-X main.GitHash=$(shell git rev-parse HEAD)
COMPOSE := IMAGE_TAG=$(IMAGE_TAG) CASSEM_EXAMPLES_DIR=$(abspath examples) $(CONTAINER_TOOL) compose -p cassem -f examples/compose.cluster.yaml

define CLUSTER_ENDPOINTS
	@echo ""
	@echo "Endpoints:"
	@echo "  cassemdb: 127.0.0.1:2021,127.0.0.1:2022,127.0.0.1:2023"
	@echo "  cassemadm: http://127.0.0.1:20218/ui (using superadmin@example.com/cassem to visit UI page)"
	@echo "  cassemagent: 127.0.0.1:20219"
endef

.PHONY: help build-cli build-image ui.install ui.build ui.test ui.lint ui.typecheck cluster.up cluster.start cluster.stop cluster.restart cluster.status cluster.logs cluster.clean test test.integration.cluster lint vet

help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Cluster:"
	@echo "  build-cli            Build local cassemdb-viewer CLI"
	@echo "  build-image          Build Linux binaries and local container images"
	@echo "  cluster.start        Build images and start the local Compose cluster"
	@echo "  cluster.stop         Stop the local Compose cluster"
	@echo "  cluster.restart      Restart the local Compose cluster"
	@echo "  cluster.status       Show local Compose cluster status"
	@echo "  cluster.logs         Show local Compose cluster logs"
	@echo "  cluster.clean        Stop cluster, remove volumes, and remove generated binaries"
	@echo "  test.integration.cluster  Build, start, wait, test, log, and clean the integration cluster"
	@echo ""
	@echo "Quality:"
	@echo "  ui.install           Install cassemadm Web dependencies"
	@echo "  ui.build             Build embedded cassemadm Web assets"
	@echo "  ui.test              Run cassemadm Web tests"
	@echo "  ui.lint              Run cassemadm Web lint"
	@echo "  ui.typecheck         Run cassemadm Web typecheck"
	@echo "  test                 Run all Go tests"
	@echo "  lint                 Run golangci-lint"
	@echo "  vet                  Run go vet"

ui.install:
	$(MAKE) -C web install

ui.build:
	$(MAKE) -C web build

ui.test:
	$(MAKE) -C web test

ui.lint:
	$(MAKE) -C web lint

ui.typecheck:
	$(MAKE) -C web typecheck

build-cli:
	mkdir -p ./bin
	go build -o ./bin/cassemdb-viewer -ldflags "$(LDFLAGS)" ./cmd/cassemdb-viewer

build-image: ui.build
	mkdir -p ./bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/cassemdb -ldflags "$(LDFLAGS)" ./cmd/cassemdb
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/cassemadm -ldflags "$(LDFLAGS)" ./cmd/cassemadm
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/cassemagent -ldflags "$(LDFLAGS)" ./cmd/cassemagent
	$(CONTAINER_TOOL) build -t yeqown/cassemdb:$(IMAGE_TAG) -f ./.deploy/dockerfiles/cassemdb.Dockerfile .
	$(CONTAINER_TOOL) build -t yeqown/cassemadm:$(IMAGE_TAG) -f ./.deploy/dockerfiles/cassemadm.Dockerfile .
	$(CONTAINER_TOOL) build -t yeqown/cassemagent:$(IMAGE_TAG) -f ./.deploy/dockerfiles/cassemagent.Dockerfile .

cluster.up: build-image
	$(COMPOSE) up -d

cluster.start: cluster.up cluster.status

cluster.stop:
	$(COMPOSE) down

cluster.restart:
	$(MAKE) cluster.stop
	$(MAKE) cluster.start

cluster.status:
	$(COMPOSE) ps
	$(CLUSTER_ENDPOINTS)

cluster.logs:
	$(COMPOSE) logs --tail=200

cluster.clean:
	$(COMPOSE) down -v --remove-orphans
	rm -rf ./bin

test.integration.cluster: cluster.clean cluster.up
	set -e; \
	status=0; \
	trap 'status=$$?; if [ $$status -ne 0 ]; then $(MAKE) cluster.status || true; $(MAKE) cluster.logs || true; fi; $(MAKE) cluster.clean || true; exit $$status' EXIT; \
	CASSEM_INTEGRATION_STRICT=1 go test -tags integration ./tests/testutil -run TestIntegrationClusterReady -count=1 -v; \
	CASSEM_INTEGRATION_STRICT=1 go test -tags integration ./tests/integration/... -p 1 -count=1 -v

test: ui.typecheck ui.test
	go test ./...

lint: ui.typecheck ui.lint
	golangci-lint run

vet:
	go vet ./...

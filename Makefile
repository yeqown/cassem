.DEFAULT_GOAL := help

CONTAINER_TOOL ?= podman
IMAGE_TAG ?= latest

LDFLAGS := -s \
	-X main.Version=$(shell git describe --tags --abbrev=0 2>/dev/null || true) \
	-X main.BuildTime=$(shell TZ=UTC date -u '+%Y-%m-%dT%H:%M:%SZ') \
	-X main.GitHash=$(shell git rev-parse HEAD)
COMPOSE := IMAGE_TAG=$(IMAGE_TAG) CASSEM_EXAMPLES_DIR=$(abspath examples) $(CONTAINER_TOOL) compose -p cassem -f examples/compose.cluster.yaml

.PHONY: help build-image ui.install ui.build ui.test ui.lint ui.typecheck cluster.start cluster.stop cluster.restart cluster.status cluster.logs cluster.clean test test.integration lint vet

help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Cluster:"
	@echo "  build-image          Build Linux binaries and local container images"
	@echo "  cluster.start        Build images and start the local Compose cluster"
	@echo "  cluster.stop         Stop the local Compose cluster"
	@echo "  cluster.restart      Restart the local Compose cluster"
	@echo "  cluster.status       Show local Compose cluster status"
	@echo "  cluster.logs         Show local Compose cluster logs"
	@echo "  cluster.clean        Stop cluster, remove volumes, and remove generated binaries"
	@echo ""
	@echo "Quality:"
	@echo "  ui.install           Install cassemadm Web dependencies"
	@echo "  ui.build             Build embedded cassemadm Web assets"
	@echo "  ui.test              Run cassemadm Web tests"
	@echo "  ui.lint              Run cassemadm Web lint"
	@echo "  ui.typecheck         Run cassemadm Web typecheck"
	@echo "  test                 Run all Go tests"
	@echo "  test.integration     Run integration tests against local cluster"
	@echo "  lint                 Run golangci-lint"
	@echo "  vet                  Run go vet"

internal/cassemadm/web/node_modules/.deps-installed: internal/cassemadm/web/package.json internal/cassemadm/web/package-lock.json
	npm ci --prefix internal/cassemadm/web
	touch internal/cassemadm/web/node_modules/.deps-installed

ui.install: internal/cassemadm/web/node_modules/.deps-installed

ui.build: internal/cassemadm/web/node_modules/.deps-installed
	npm run build --prefix internal/cassemadm/web

ui.test: internal/cassemadm/web/node_modules/.deps-installed
	npm run test:run --prefix internal/cassemadm/web

ui.lint: internal/cassemadm/web/node_modules/.deps-installed
	npm run lint --prefix internal/cassemadm/web

ui.typecheck: internal/cassemadm/web/node_modules/.deps-installed
	npm run typecheck --prefix internal/cassemadm/web

build-image: ui.build
	mkdir -p ./bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/cassemdb -ldflags "$(LDFLAGS)" ./cmd/cassemdb
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/cassemadm -ldflags "$(LDFLAGS)" ./cmd/cassemadm
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/cassemagent -ldflags "$(LDFLAGS)" ./cmd/cassemagent
	$(CONTAINER_TOOL) build -t yeqown/cassemdb:$(IMAGE_TAG) -f ./.deploy/dockerfiles/cassemdb.Dockerfile .
	$(CONTAINER_TOOL) build -t yeqown/cassemadm:$(IMAGE_TAG) -f ./.deploy/dockerfiles/cassemadm.Dockerfile .
	$(CONTAINER_TOOL) build -t yeqown/cassemagent:$(IMAGE_TAG) -f ./.deploy/dockerfiles/cassemagent.Dockerfile .

cluster.start: build-image
	$(COMPOSE) up -d
	$(COMPOSE) ps
	@echo ""
	@echo "Endpoints:"
	@echo "  cassemdb: 127.0.0.1:2021,127.0.0.1:2022,127.0.0.1:2023"
	@echo "  cassemadm: http://127.0.0.1:20218"
	@echo "  cassemagent: 127.0.0.1:20219"

cluster.stop:
	$(COMPOSE) down

cluster.restart:
	$(MAKE) cluster.stop
	$(MAKE) cluster.start

cluster.status:
	$(COMPOSE) ps
	@echo ""
	@echo "Endpoints:"
	@echo "  cassemdb: 127.0.0.1:2021,127.0.0.1:2022,127.0.0.1:2023"
	@echo "  cassemadm: http://127.0.0.1:20218/ui (using superadmin@example.com/cassem to visit UI page)"
	@echo "  cassemagent: 127.0.0.1:20219"

cluster.logs:
	$(COMPOSE) logs --tail=200

cluster.clean:
	$(COMPOSE) down -v --remove-orphans
	rm -rf ./bin

test: ui.typecheck ui.test
	go test ./...

test.integration:
	go test -tags integration ./tests/integration/... -count=1

lint: ui.typecheck ui.lint
	golangci-lint run

vet:
	go vet ./...

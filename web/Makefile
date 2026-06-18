.DEFAULT_GOAL := help

NODE_MODULES_STAMP := node_modules/.deps-installed

.PHONY: help install build test lint typecheck

help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "UI:"
	@echo "  install     Install dependencies"
	@echo "  build       Build assets"
	@echo "  test        Run tests"
	@echo "  lint        Run lint"
	@echo "  typecheck   Run typecheck"

$(NODE_MODULES_STAMP): package.json package-lock.json
	npm ci
	touch $(NODE_MODULES_STAMP)

install: $(NODE_MODULES_STAMP)

build: $(NODE_MODULES_STAMP)
	npm run build

test: $(NODE_MODULES_STAMP)
	npm run test:run

lint: $(NODE_MODULES_STAMP)
	npm run lint

typecheck: $(NODE_MODULES_STAMP)
	npm run typecheck

# Convenience targets for local demo and smoke tests.
#
# The control-plane and UI are started in separate terminals by design.
# Set EXTRACTION_CRAWL4AI_PYTHON to a venv Python for the full extraction ok path.

.PHONY: help demo-api demo-ui smoke smoke-ok smoke-platform smoke-api smoke-crm install-extraction-venv test-go test-ui

help:
	@echo "Available targets:"
	@echo "  make test-go             - run all Go module tests + control-plane tests"
	@echo "  make test-ui             - run UI typecheck, lint, and build"
	@echo "  make demo-api            - start control-plane on :8080"
	@echo "  make demo-api-ok         - start control-plane with extraction venv"
	@echo "  make demo-ui             - start Next.js UI on :3000"
	@echo "  make smoke               - run smoke-extraction.sh (API must be up)"
	@echo "  make smoke-ok            - run smoke and require extraction ok/partial"
	@echo "  make smoke-platform      - run extraction + email-validate smoke (API must be up)"
	@echo "  make smoke-api           - run operator smoke against localhost:8080 (API must be up)"
	@echo "  make smoke-crm           - run crm_ready promote/export smoke (API must be up)"
	@echo "  make install-extraction-venv - create modules/extraction/.venv with Crawl4AI"

demo-api:
	@echo "Starting control-plane on http://localhost:8080"
	@cd services/control-plane && go run ./cmd/server

demo-api-ok:
	@if [ -z "${EXTRACTION_CRAWL4AI_PYTHON}" ]; then \
		echo "EXTRACTION_CRAWL4AI_PYTHON is not set."; \
		echo "Run 'make install-extraction-venv' and then re-run with:"; \
		echo "  EXTRACTION_CRAWL4AI_PYTHON=$(PWD)/modules/extraction/.venv/bin/python make demo-api-ok"; \
		exit 1; \
	fi
	@echo "Starting control-plane with extraction venv: ${EXTRACTION_CRAWL4AI_PYTHON}"
	@cd services/control-plane && EXTRACTION_CRAWL4AI_PYTHON="${EXTRACTION_CRAWL4AI_PYTHON}" go run ./cmd/server

demo-ui:
	@echo "Starting Next.js UI on http://localhost:3000"
	@cd ui/web-console && npm run dev

smoke:
	@./scripts/smoke-extraction.sh

smoke-ok:
	@SMOKE_REQUIRE_OK=1 ./scripts/smoke-extraction.sh

smoke-platform:
	@./scripts/smoke-platform.sh

install-extraction-venv:
	cd modules/extraction && python3 -m venv .venv
	. modules/extraction/.venv/bin/activate && pip install -r modules/extraction/requirements.txt
	@echo "Optional: install Playwright browsers if Crawl4AI needs them"
	@. modules/extraction/.venv/bin/activate && python -m playwright install chromium || true
	@echo "Run the API with: EXTRACTION_CRAWL4AI_PYTHON=$(CURDIR)/modules/extraction/.venv/bin/python make demo-api-ok"

test-go:
	@set -e; \
	for m in modules/*; do \
		if [ -f "$$m/go.mod" ]; then \
			echo "==> $$m"; \
			cd "$$m" && go test -short ./... && go test ./... && go vet ./... && go build ./...; \
			cd - > /dev/null; \
		fi; \
	done; \
	echo "==> services/control-plane"; \
	cd services/control-plane && go test ./... && go vet ./... && go build ./...

test-ui:
	@cd ui/web-console && npm run typecheck && npm run lint && npm run build

smoke-api:
	@./scripts/smoke-api.sh

smoke-crm:
	@./scripts/smoke-crm-ready.sh

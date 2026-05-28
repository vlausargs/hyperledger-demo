.PHONY: build-chaincode build-api test-chaincode test-api test test-integration test-integration-chaincode test-integration-api test-web test-e2e test-all lint clean deploy-monitoring stop-monitoring deploy-api deploy-web deploy-proxy deploy-all stop-api stop-web stop-proxy stop-all config-validate config-policy test-config deploy-fabric stop-fabric add-org helm-lint

build-chaincode:
	cd packages/chaincode && go build -o ../../bin/chaincode .

build-api:
	cd packages/api && go build -o ../../bin/server ./cmd/server

build: build-chaincode build-api

test-chaincode:
	cd packages/chaincode && go test ./... -v -race -cover

test-api:
	cd packages/api && go test ./... -v -race -cover

test: test-chaincode test-api

# ─── Integration & E2E ──────────────────────────────────────────────────────

# Chaincode integration tests — multi-contract supply chain flows.
test-integration-chaincode:
	cd packages/chaincode && go test -tags=integration -v -race ./contracts/...

# API integration tests — full HTTP stack with mock FabricGateway.
test-integration-api:
	cd packages/api && go test -tags=integration -v -race ./test/integration/...

# All integration tests (every PR, <5 min target per spec section 10).
test-integration: test-integration-chaincode test-integration-api

# Frontend vitest suite. Runs once and exits.
test-web:
	cd packages/web && pnpm test --run

# E2E Playwright suite. Requires the full stack to be running.
# Use E2E_BASE_URL=... and E2E_API_HEALTH=... to point at a non-default host.
test-e2e:
	cd e2e/playwright && pnpm exec playwright test

# Run everything: unit + integration + web + e2e.
test-all: test test-integration test-web test-e2e

lint-chaincode:
	cd packages/chaincode && go vet ./...

lint-api:
	cd packages/api && go vet ./...

lint: lint-chaincode lint-api

clean:
	rm -f bin/chaincode bin/server

deploy-monitoring:
	docker compose -f infra/docker/compose.monitoring.yml up -d

stop-monitoring:
	docker compose -f infra/docker/compose.monitoring.yml down

deploy-api:
	docker compose -f infra/docker/compose.api.yml up -d --build

deploy-web:
	docker compose -f infra/docker/compose.web.yml up -d --build

deploy-proxy:
	docker compose -f infra/docker/compose.proxy.yml up -d

deploy-fabric:
	docker compose -f infra/docker/compose.fabric.yml up -d

stop-fabric:
	docker compose -f infra/docker/compose.fabric.yml down

deploy-all: deploy-fabric deploy-api deploy-web deploy-proxy deploy-monitoring

stop-api:
	docker compose -f infra/docker/compose.api.yml down

stop-web:
	docker compose -f infra/docker/compose.web.yml down

stop-proxy:
	docker compose -f infra/docker/compose.proxy.yml down

stop-all: stop-api stop-web stop-proxy stop-monitoring stop-fabric

config-validate:
	cd infra/configloader && go run ./cmd/main.go validate

config-policy:
	cd infra/configloader && go run ./cmd/main.go policy

test-config:
	cd infra/configloader && go test ./... -v

# add-org ORG=org4 -- provision a new organization end-to-end.
add-org:
	@if [ -z "$(ORG)" ]; then echo "usage: make add-org ORG=<name>" >&2; exit 1; fi
	ORG=$(ORG) bash fabric-network/scripts/deployment/010-add-org.sh

helm-lint:
	@for chart in infra/k8s/helm/*/; do \
		echo "==> helm lint $$chart"; \
		helm lint $$chart || exit 1; \
	done

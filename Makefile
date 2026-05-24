.PHONY: build-chaincode build-api test-chaincode test-api test lint clean deploy-monitoring stop-monitoring deploy-api deploy-web deploy-proxy deploy-all stop-api stop-web stop-proxy stop-all config-validate config-policy test-config

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

deploy-all: deploy-api deploy-web deploy-proxy deploy-monitoring

stop-api:
	docker compose -f infra/docker/compose.api.yml down

stop-web:
	docker compose -f infra/docker/compose.web.yml down

stop-proxy:
	docker compose -f infra/docker/compose.proxy.yml down

stop-all: stop-api stop-web stop-proxy stop-monitoring

config-validate:
	cd infra/configloader && go run ./cmd/main.go validate

config-policy:
	cd infra/configloader && go run ./cmd/main.go policy

test-config:
	cd infra/configloader && go test ./... -v

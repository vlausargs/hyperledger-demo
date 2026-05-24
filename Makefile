.PHONY: build-chaincode build-api test-chaincode test-api test lint clean deploy-monitoring stop-monitoring

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

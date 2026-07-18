# Chaincode-as-a-Service (ccaas) image for the `basic` supply-chain chaincode.
#
# packages/chaincode/main.go calls contractapi.NewChaincode(...).Start(), and
# contract-api-go v2 (contractapi.(*ContractChaincode).Start(), see
# fabric-contract-api-go/v2@v2.0.0/contractapi/contract_chaincode.go) already
# branches into a ccaas gRPC server (shim.ChaincodeServer, a real TCP
# listener) whenever CHAINCODE_SERVER_ADDRESS and CORE_CHAINCODE_ID_NAME are
# both set in the environment, falling back to the peer-launched
# shim.Start(cc) stdio path only when either is unset. No source change is
# needed — this Dockerfile only needs to set CHAINCODE_SERVER_ADDRESS;
# CORE_CHAINCODE_ID_NAME is injected per-deployment by
# `kubectl hlf externalchaincode sync --package-id=...` (it varies with the
# computed package-id, so it can't be baked into the image).
#
# Port 7052: `kubectl hlf externalchaincode sync` always creates the
# in-cluster Service on port 7052 -> targetPort 7052, regardless of its
# `--port` flag (empirically confirmed on this cluster: that flag only
# affects the Deployment's CHAINCODE_SERVER_ADDRESS env value and
# liveness/readiness tcpSocket probe port, not the Service). The Deployment
# env CHAINCODE_SERVER_ADDRESS set by kubectl-hlf overrides this image's ENV
# at container-start (Kubernetes pod-spec env wins over the image default),
# so 7052 here is both the functional fallback for standalone/non-k8s runs
# and, so long as `sync` is invoked without an explicit `--port` (leaving it
# at its 7052 default, as 50-chaincode.sh does), the value actually used
# in-cluster too -- keeping image default and Service port aligned avoids a
# silent mismatch if the deploy script's flags ever change.
FROM golang:1.23 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /chaincode .

FROM gcr.io/distroless/static-debian12
COPY --from=build /chaincode /chaincode
ENV CHAINCODE_SERVER_ADDRESS=0.0.0.0:7052
EXPOSE 7052
ENTRYPOINT ["/chaincode"]

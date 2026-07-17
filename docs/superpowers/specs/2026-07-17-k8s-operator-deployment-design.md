# K8s Operator Deployment — Design (Sub-project 1)

**Date:** 2026-07-17
**Status:** Approved, pending implementation plan
**Scope:** Get the existing supply-chain Fabric network running on a local **kind** cluster via the **hlf-operator**, with the existing `api` and `web` connected end-to-end.

---

## Context

Today the network runs on bare-metal via numbered bash scripts (`fabric-network/scripts/deployment/001–010`), everything on `localhost`. The existing Helm charts in `infra/k8s/helm/fabric/` are **skeletons** — pods, ports, PVCs only, with no crypto/MSP/Secret/bootstrap wiring — so they render but produce a non-functional Fabric.

The goal is production Kubernetes deployment. This sub-project is the **local rehearsal**: prove the operator-based path on kind, staying as close to prod as possible (same manifests kind → prod; only the node substrate differs — containers locally, real machines in prod).

### Decisions already made (see memory `project_k8s_deployment_goal`)

- **Operator over hand-rolled Helm.** Declarative CRDs, self-healing, cert rotation built in; becomes the backend for the future UI.
- **hlf-operator** (`hyperledger-labs/hlf-operator`, aka bevel-operator-fabric, Kung Fu Software) over `fabric-operator`. Complete CRDs (CA/peer/orderer/channel/chaincode), Fabric 2.3–3.x, mature `kubectl-hlf` plugin. Chosen because the user will build their **own** SvelteKit admin UI later (not reuse fabric-operator's bundled Console), and without that Console fabric-operator's CRDs are incomplete.
- **Future UI pattern:** SvelteKit → k8s API (apply CRD) → hlf-operator → Fabric. 3 layers, no Console backend.

### The larger vision, decomposed

| Sub-project | What | When |
|---|---|---|
| **1. Network on kind via hlf-operator** | This doc | now |
| 2. Custom admin UI | SvelteKit → k8s API to add org/channel/cc dynamically | later |
| 3. Real multi-server prod | Same CRDs on cloud k8s; Istio for cross-cluster mTLS | later |

Each gets its own spec → plan → build. This doc is **Sub-project 1 only**.

---

## Target network (mirrors current bare-metal network)

| Concept | Value |
|---|---|
| Peer orgs | Org1 (`Org1MSP`, manufacturer), Org2 (`Org2MSP`, distributor) |
| Orderer org | `OrdererMSP` |
| CAs | `org1-ca`, `org2-ca`, `ord-ca` |
| Orderer nodes | 1 (kind). **Prod: 3–5** for Raft crash-fault-tolerance |
| Peers | `peer0-org1`, `peer0-org2`, each with a CouchDB state DB |
| Channel | `mychannel` |
| Chaincode | `basic` — built from `packages/chaincode` (Go), deployed as ccaas |
| Endorsement policy | `AND(Org1MSP, Org2MSP)` (unchanged) |
| Fabric version | 2.5.15 (operator supports through 3.1) |

---

## Architecture

```
kind cluster (single node is sufficient)
└── namespace: hlf
    ├── hlf-operator                 (helm: kfs/hlf-operator v1.13.x)
    │      watches CRDs, reconciles pods, generates crypto
    │
    ├── FabricCA ×3                  org1-ca, org2-ca, ord-ca
    ├── FabricOrdererNode ×1         orderer0            (prod: 3+)
    ├── FabricPeer ×2                peer0-org1, peer0-org2  (+ CouchDB each)
    ├── FabricMainChannel            mychannel           (create)
    ├── FabricFollowerChannel ×2     peer0-org1, peer0-org2 join
    ├── FabricExternalChaincode      basic  (ccaas pod, image from packages/chaincode)
    │
    └── api ×2 + web                 (existing Helm charts, connection profile → in-cluster peers)
```

**Substrate parity:** identical CRD manifests deploy on prod k8s. Only difference kind → prod = real nodes, real StorageClass, real DNS/ingress (Sub-project 3).

---

## Components & build order

1. **Prereqs (host):** `kind`, `kubectl`, `helm`, krew, and the `kubectl-hlf` plugin (`kubectl krew install hlf`). A kind config with port mappings for reaching NodePort services.
2. **Cluster + operator:**
   ```
   kind create cluster --config infra/k8s/operator/kind-config.yaml
   helm repo add kfs https://kfsoftware.github.io/hlf-helm-charts
   helm install hlf-operator --version=1.13.0 kfs/hlf-operator -n hlf --create-namespace
   ```
3. **CAs:** create 3 `FabricCA` (via `kubectl hlf ca create` or CRD manifests). Register + enroll admin/peer/orderer identities — operator handles crypto, no manual `fabric-ca-client`.
4. **Orderer:** 1 `FabricOrdererNode` (`kubectl hlf ordnode create`), MSP `OrdererMSP`.
5. **Peers:** 2 `FabricPeer` (`kubectl hlf peer create`), CouchDB state DB, TLS enabled.
6. **Channel:** `FabricMainChannel` (`mychannel`, both orgs + orderer) → 2 `FabricFollowerChannel` to join peers.
7. **Chaincode:** build `packages/chaincode` (Go) into a container image, load into kind, deploy as `FabricExternalChaincode` via `kubectl hlf externalchaincode sync`; approve for both orgs; commit. Endorsement `AND(Org1MSP, Org2MSP)`.
8. **Apps:** regenerate the Gateway connection profile with in-cluster peer/orderer DNS + embedded TLS CA certs; deploy `api` + `web` via existing Helm charts; expose via NodePort/port-forward.
9. **Verify:** see Verification.

Repo additions:
- `infra/k8s/operator/` — kind config + CRD manifests (or a documented `kubectl-hlf` command script).
- `Makefile` — `kind-up`, `kind-down`, `kind-network`, `kind-chaincode`, `kind-apps` targets, mirroring the existing `network-*` group.

---

## Key design choices

- **Skip Istio on kind.** hlf-operator's quickstart uses Istio for external peer access; a single-cluster kind network reaches peers over in-cluster k8s DNS and does not need it. Expose only `api`/`web` via NodePort/port-forward. Istio is reintroduced in Sub-project 3 for cross-cluster mTLS. Rationale: lighter stack, fewer new concepts for a first-time HLF developer.
- **Orderer = 1 node** on kind. Explicitly noted as **not** production-safe; prod uses 3–5 Raft nodes.
- **Chaincode as a service (ccaas).** Cleaner than peer-built chaincode images; the chaincode runs as its own pod the operator manages.
- **Retire `infra/k8s/helm/fabric/`.** The operator supersedes the skeleton fabric chart. The `api`, `web`, and `monitoring` Helm charts are kept.
- **Reuse, don't rewrite, app code.** `api`/`web` are unchanged; only the connection profile's endpoints change from `localhost` to in-cluster DNS.

---

## Data flow (unchanged from today)

`api` (Fabric Gateway SDK) → `peer0` gRPC (endorse) → collect endorsements → `orderer` (order into block) → peers validate + commit → CouchDB world state. Identical to the bare-metal flow; only the network addresses in the connection profile differ.

---

## Verification

- `kubectl get pods -n hlf` → all Running (CAs, orderer, peers, CouchDB, chaincode, api, web).
- `kubectl hlf channel inspect --channel mychannel …` → both peers joined.
- Invoke a chaincode transaction (via `kubectl hlf` or the `api`) → succeeds, endorsed by both orgs.
- Existing Playwright e2e suite runs against the `api` endpoint → passes.

**Success = all four green.**

---

## Out of scope (later sub-projects)

- Custom SvelteKit admin UI (Sub-project 2).
- Multi-server / cluster-per-org / Istio cross-cluster mTLS (Sub-project 3).
- HA orderer (3–5 nodes), production secret management, real DNS/TLS SANs, monitoring stack on k8s.

---

## Risks

| Risk | Mitigation |
|---|---|
| First-time HLF + operator + k8s at once | Go step-by-step on kind; each build-order step verified before the next. |
| ccaas image not reachable in kind | `kind load docker-image` to preload; verify chaincode pod logs. |
| Connection profile drift (localhost → DNS) | Regenerate from operator-issued material; do not hand-edit. |
| Multi-org signature collection for future add-org | Deferred to Sub-project 2; hlf-operator makes this more manual than the Console did — noted, not solved here. |
| Fabric 2.5 vs operator defaults (3.x) | Pin Fabric image tags to 2.5.15 in CRDs. |

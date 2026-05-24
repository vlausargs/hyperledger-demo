# Production Readiness Design — HLF Supply Chain

**Date:** 2026-05-24
**Status:** Draft
**Approach:** Clean Architecture Monorepo (Approach 2)

---

## Context

Hyperledger Fabric 2.5.15 supply chain network with 3 orgs (Manufacturer, Distributor, Retailer). Currently functional as a demo. Needs production hardening across: code architecture, security, CI/CD, monitoring. End goal is dynamic multi-tenant org provisioning.

### Constraints

- 2 developers (mostly solo)
- 50K+ transactions/day target
- Support both VPS and Kubernetes deployment
- GitHub + GitHub Actions for CI/CD
- pnpm for frontend package management
- First-time HLF team — architecture must be clear and self-documenting

---

## 1. Project Structure

```
hlf-supply-chain/
├── packages/
│   ├── chaincode/
│   │   ├── contracts/
│   │   │   ├── product.go
│   │   │   ├── shipment.go
│   │   │   ├── custody.go
│   │   │   ├── event.go
│   │   │   ├── recall.go
│   │   │   ├── sale.go
│   │   │   └── inventory.go
│   │   ├── models/
│   │   │   ├── product.go
│   │   │   ├── shipment.go
│   │   │   ├── custody.go
│   │   │   ├── event.go
│   │   │   ├── recall.go
│   │   │   ├── sale.go
│   │   │   └── common.go
│   │   ├── ledger/
│   │   │   └── store.go
│   │   ├── validation/
│   │   │   └── validate.go
│   │   ├── chaincode.go
│   │   ├── chaincode_test.go
│   │   ├── go.mod
│   │   └── META-INF/statedb/couchdb/indexes/
│   │
│   ├── api/
│   │   ├── cmd/
│   │   │   └── server/main.go
│   │   ├── internal/
│   │   │   ├── config/
│   │   │   ├── handler/
│   │   │   │   ├── product.go
│   │   │   │   ├── shipment.go
│   │   │   │   ├── custody.go
│   │   │   │   ├── event.go
│   │   │   │   ├── recall.go
│   │   │   │   ├── pos.go
│   │   │   │   ├── network.go
│   │   │   │   └── auth.go
│   │   │   ├── middleware/
│   │   │   ├── service/
│   │   │   │   ├── fabric.go
│   │   │   │   └── ca.go
│   │   │   └── router/
│   │   ├── pkg/
│   │   │   ├── errors/
│   │   │   └── response/
│   │   └── go.mod
│   │
│   ├── web/
│   │   ├── src/
│   │   │   ├── lib/
│   │   │   │   ├── api/
│   │   │   │   ├── components/
│   │   │   │   └── stores/
│   │   │   └── routes/
│   │   │       ├── (auth)/
│   │   │       │   └── login/
│   │   │       ├── (app)/
│   │   │       │   ├── dashboard/
│   │   │       │   ├── manufacturer/
│   │   │       │   ├── distributor/
│   │   │       │   ├── retailer/
│   │   │       │   ├── trace/
│   │   │       │   ├── recalls/
│   │   │       │   └── admin/
│   │   │       └── +layout.server.ts
│   │   ├── package.json
│   │   ├── pnpm-lock.yaml
│   │   └── svelte.config.js
│   │
│   └── shared/
│       └── types/
│
├── infra/
│   ├── docker/
│   │   ├── compose.fabric.yml
│   │   ├── compose.api.yml
│   │   ├── compose.web.yml
│   │   ├── compose.monitoring.yml
│   │   ├── compose.proxy.yml
│   │   ├── compose.override.yml
│   │   └── Dockerfiles/
│   ├── k8s/
│   │   └── helm/
│   ├── scripts/
│   │   ├── deploy/
│   │   └── helpers/
│   └── config/
│       ├── network.yml
│       ├── orgs/
│       │   ├── org1.yml
│       │   ├── org2.yml
│       │   └── org3.yml
│       ├── environments/
│       │   ├── dev.yml
│       │   ├── staging.yml
│       │   └── production.yml
│       ├── configtx.yaml
│       ├── core.yaml
│       └── connection-profiles/
│
├── .github/
│   └── workflows/
│       ├── ci.yml
│       ├── build.yml
│       └── deploy.yml
│
├── .env.example
├── Makefile
├── RULES.md
└── docs/
    ├── architecture.md
    └── runbook.md
```

Key changes from current layout:
- Monolithic `chaincode.go` (1,313 lines) split into 7 contract files + models + ledger layer + validation
- Monolithic `handlers.go` (672 lines) split by domain with service layer between handlers and Fabric
- `internal/` for unexported Go packages (standard Go convention)
- Config validation at startup (currently missing)
- Shared types between Go API and TypeScript frontend
- POS app (`pos-app/`) removed — replaced by `packages/web/`

---

## 2. Chaincode Architecture

### Contract Pattern

Each domain gets its own contract struct implementing `contractapi.ContractInterface`:

```go
type ProductContract struct {
    contractapi.Contract
    store     *ledger.Store
    validator *validation.Validator
}

func (c *ProductContract) CreateProduct(ctx contractapi.TransactionContextInterface, input string) error {
    // 1. Parse + validate input
    // 2. Check caller MSP (config-driven, not hardcoded)
    // 3. Store via ledger layer
    // 4. Emit event
}
```

### Ledger Abstraction Layer

`ledger/store.go` wraps `stub.PutState`/`GetState`:
- Key generation (prefix + composite keys)
- JSON marshal/unmarshal
- Pagination helpers
- CouchDB query builder (parameterized selectors — prevents injection)

### Config-Driven Access Control

Replace hardcoded MSP IDs (`"Org1MSP"`, `"Org3MSP"`) with configurable rules:

```go
var AccessRules = map[string][]string{
    "CreateProduct":  {"ManufacturerMSP"},
    "CreateSale":     {"RetailerMSP"},
    "CreateShipment": {"ManufacturerMSP", "DistributorMSP"},
}
```

### Input Validation

`validation/validate.go`:
- ID format: alphanumeric + `_-`, max 64 chars
- Required fields
- Status transitions (state machine — ACTIVE->SHIPPED allowed, SHIPPED->ACTIVE rejected)
- CouchDB query parameter sanitization

### Testing

Mock `ChaincodeStubInterface` for unit tests. Each contract tested independently. Target: 80%+ coverage on business logic.

---

## 3. REST API Architecture

### Clean Architecture (3 layers)

```
HTTP Request
    |
[Middleware] -> JWT, CORS, Rate Limit, Request ID, Structured Logging
    |
[Handler]   -> Parse request, validate input, call service, format response
    |
[Service]   -> Business logic, Fabric Gateway calls, error mapping
    |
[Fabric]    -> gRPC connection, identity management, contract access
```

### Config Module

`internal/config/config.go`:
- Parse all env vars / YAML config at startup with validation
- Fail fast with clear error if required vars missing
- Typed config struct (no `os.Getenv` scattered through code)

### Service Layer

Handlers never touch Fabric directly:

```go
// handler — thin
func (h *ProductHandler) Create(c *gin.Context) {
    var req CreateProductRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, err)
        return
    }
    product, err := h.productService.Create(c.Request.Context(), req)
}

// service — business logic
func (s *ProductService) Create(ctx context.Context, req CreateProductRequest) (*Product, error) {
    // validate, call fabric, map errors
}
```

### Middleware Stack

- **Rate limiter** — token bucket per IP and per user (critical at 50K+ txns/day)
- **Request timeout** — per-endpoint configurable
- **Structured logging** — JSON logs with request ID, org, user, duration
- **Request ID propagation** — accept `X-Request-ID` header or generate, pass to Fabric

### Standard Error Responses

```go
type AppError struct {
    Code    int    // HTTP status
    Message string // User-facing
    Detail  string // Internal (logged, not returned)
}
```

### Health Checks

- `/health/live` — process alive (K8s liveness probe)
- `/health/ready` — Fabric Gateway connected, can submit txns (K8s readiness probe)

### Connection Management (50K+ txns/day)

- Single Fabric Gateway connection per org (reused across requests)
- gRPC keepalive tuning
- Graceful shutdown with in-flight request draining

---

## 4. Frontend Architecture

**Framework:** SvelteKit + Tailwind CSS + shadcn-svelte
**Package manager:** pnpm
**Auth:** Server-side JWT in httpOnly cookies (not localStorage)

### Route Structure (Role-Based Views)

```
src/routes/
├── (auth)/
│   ├── login/+page.svelte
│   └── +layout.server.ts
│
├── (app)/+layout.server.ts        # Auth guard
├── (app)/+layout.svelte           # Sidebar nav (adapts to org role)
│
├── (app)/dashboard/               # Stats, recent activity, alerts
│
├── (app)/manufacturer/            # Org1 only
│   ├── products/                  # Create, manage, batch tracking
│   └── shipments/                 # Create outbound shipments
│
├── (app)/distributor/             # Org2 only
│   ├── receiving/                 # Accept/reject custody transfers
│   ├── inventory/                 # Current stock
│   └── shipments/                 # Forward to retailer
│
├── (app)/retailer/                # Org3 only
│   ├── pos/                       # Point of sale (checkout, barcode scan)
│   ├── inventory/                 # Stock levels
│   └── sales/                     # Sales history, receipts
│
├── (app)/trace/                   # All orgs
│   └── [productId]/               # Full provenance
│
├── (app)/recalls/                 # All orgs
│
└── (app)/admin/                   # Network ops (future: multi-tenant org management)
    ├── network/
    └── identities/
```

### Auth Flow

1. User logs in -> Go API returns JWT
2. SvelteKit server receives it, sets `Set-Cookie: token=xxx; HttpOnly; Secure; SameSite=Strict`
3. Browser stores cookie automatically, sends on every request
4. JavaScript never touches the token

### API Proxy

Browser never calls Go API directly:

```
Browser -> SvelteKit server (+page.server.ts) -> Go REST API
```

Benefits: hide API URLs from browser, server-side caching, token refresh handling.

### Role Guard

`+layout.server.ts` checks org role, redirects if user accesses wrong section.

### Real-Time Updates

SSE (Server-Sent Events) from Go API for:
- New custody transfer requests
- Recall alerts
- Shipment status changes

### Future: Offline Resilience

Service worker for POS checkout queue. If network drops, queue sales locally, sync when back online.

---

## 5. Security Hardening

### Input Validation

- REST API: validate all IDs, query params, request bodies before hitting chaincode
- Chaincode: validate again (defense in depth)
- CouchDB selectors: parameterized query builder, never string concat user input

### Authentication & Authorization

- JWT with short expiry (15 min access token + refresh token rotation)
- Role claims in JWT: `{ org: "Org1MSP", role: "admin|user" }`
- Per-endpoint authorization middleware
- Rate limit per user, not just per IP

### Secrets Management

- `.env` for dev only
- Production: Docker secrets or OpenBao (open-source Vault fork, MPL 2.0, Linux Foundation)
- Never log secrets — structured logger with field redaction
- Rotate JWT secret without downtime (support 2 active secrets during rotation)

### TLS

- Fabric components: already TLS (keep)
- REST API: HTTPS termination via Caddy reverse proxy (auto Let's Encrypt)
- CouchDB: TLS for multi-machine deployment

### API Security Headers

```
Strict-Transport-Security: max-age=31536000
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Content-Security-Policy: default-src 'self'
```

### Audit Logging

- Every state-changing API call logged: who, what, when, from where
- Separate audit log stream (not mixed with app logs)
- Append-only log destination

### Dependency Scanning

- `govulncheck` in CI for Go
- `pnpm audit` in CI for frontend
- Dependabot or Renovate for automated security updates

---

## 6. Observability

**Stack:** Grafana + Prometheus + Loki (open source, self-hosted, works on VPS and K8s)

### Metrics (Prometheus)

- **Go API:** `/metrics` endpoint via `prometheus/client_golang`
  - Request rate, latency (p50/p95/p99), error rate per endpoint
  - Active Fabric Gateway connections
  - JWT auth failures, rate limit hits
- **Fabric peers:** native Prometheus metrics (enable in core.yaml)
  - Block height, endorsement count, chaincode execution time
  - Gossip peer count, ledger size
- **CouchDB:** Prometheus exporter for query latency, cache hit rate
- **PostgreSQL:** `postgres_exporter` for connection count, query duration
- **Host:** `node_exporter` for CPU, memory, disk, network

### Logging (Loki)

All Go services output structured JSON logs:

```json
{
  "level": "info",
  "ts": "2026-05-24T10:00:00Z",
  "requestId": "abc123",
  "org": "Org1MSP",
  "method": "POST",
  "path": "/api/v1/products",
  "duration_ms": 45,
  "status": 201
}
```

Loki collects, indexes by labels (org, service, level). Grafana queries logs alongside metrics.

### Alerting (Grafana Alerting)

- Fabric peer unreachable -> page
- Error rate > 5% for 5 min -> alert
- Chaincode execution p95 > 5s -> alert
- Disk usage > 80% -> warning
- CouchDB query latency spike -> warning
- Certificate expiry < 30 days -> warning

### Dashboards

1. Network health — all peers, orderer, CAs status
2. API performance — request rate, latency, errors by endpoint
3. Supply chain ops — products created/day, shipments in transit, pending custody transfers
4. Resource usage — CPU/memory/disk per container

### Deployment

Single `compose.monitoring.yml` — Prometheus + Loki + Grafana. ~512MB RAM total.

---

## 7. CI/CD Pipeline (GitHub Actions)

### `ci.yml` — Every PR

```
Jobs (parallel):
├── chaincode:
│   ├── go vet + golangci-lint
│   ├── go test -race -cover (target 80%+)
│   └── govulncheck
│
├── api:
│   ├── go vet + golangci-lint
│   ├── go test -race -cover
│   └── govulncheck
│
├── web:
│   ├── pnpm lint (eslint + svelte-check)
│   ├── pnpm test (vitest)
│   ├── pnpm build (type-check + build)
│   └── pnpm audit
│
└── infra:
    └── shellcheck on all .sh scripts
```

PR blocked until all pass.

### `build.yml` — Merge to Main

```
Jobs:
├── Run full CI
├── Build Docker images:
│   ├── api -> ghcr.io/myindo/hlf-api:sha-xxxxx
│   └── web -> ghcr.io/myindo/hlf-web:sha-xxxxx
├── Push to GitHub Container Registry
└── Tag release (semantic versioning)
```

### `deploy.yml` — Manual Trigger or Release Tag

```
Inputs: environment (staging/production)

Jobs:
├── Pull images on target server
├── Rolling restart of API containers
├── Deploy frontend
├── Smoke test (/health/ready)
└── Notify (Slack/Discord/email)
```

### Chaincode Deployment

Stays manual — chaincode lifecycle (package/install/approve/commit) requires multi-org coordination. Script-assisted but human-triggered.

### Branch Strategy

- `main` — always deployable
- `feature/*` — development branches
- `release/*` — optional, for release candidates
- PR required to merge to main, CI must pass

---

## 8. Deployment Architecture

### Docker-First (Same Artifacts for VPS and K8s)

```
                    Reverse Proxy (Caddy — auto HTTPS)
                    api.example.com -> :8080
                    app.example.com -> :3000
                    grafana.example.com -> :3001
                              |
           ┌──────────────────┼──────────────────┐
           |                  |                  |
       Web (SvelteKit)   API (Go x3)      Monitoring
       :3000             per org          Prometheus/Loki/Grafana
                              |
                    Fabric Network
                    Orderer + 3 Peers
                    4 CAs + CouchDBs + PostgreSQL
```

### Compose File Split

```
infra/docker/
├── compose.fabric.yml       # Peers, orderer, CAs, CouchDB, PostgreSQL
├── compose.api.yml          # 3 API instances (one per org)
├── compose.web.yml          # SvelteKit frontend
├── compose.monitoring.yml   # Prometheus + Loki + Grafana
├── compose.proxy.yml        # Caddy reverse proxy
└── compose.override.yml     # Dev-only overrides
```

### VPS Deployment

- `Makefile` commands: `make deploy-staging`, `make deploy-prod`
- SSH + docker compose pull + restart
- Caddy handles TLS (auto Let's Encrypt)

### K8s Deployment (Future-Ready)

- Helm charts in `infra/k8s/helm/`
- Same Docker images from GHCR
- One chart per component (fabric, api, web, monitoring)
- HPA on API pods for 50K+ txns/day

### Multi-Machine Deployment

Each server runs only its org's components, controlled by config:

```
Server A (Orderer): orderer, CA-orderer, PostgreSQL-orderer
Server B (Org1):    peer, CA, CouchDB, PostgreSQL, API
Server C (Org2):    peer, CA, CouchDB, PostgreSQL, API
Server D (Org3):    peer, CA, CouchDB, PostgreSQL, API
```

Cross-machine communication via `*_EXTERNAL_HOST` in per-org config. Firewall must open required ports.

### Makefile

```makefile
deploy-network       # Run deployment scripts (001-008)
deploy-api           # Build + start API containers
deploy-web           # Build + start frontend
deploy-monitoring    # Start Prometheus + Loki + Grafana
deploy-all           # Everything
teardown             # 999-teardown.sh
test                 # Run all tests (chaincode + api + web)
lint                 # Run all linters
logs                 # Tail all service logs
add-org ORG=orgN     # Provision new organization
```

---

## 9. Config Management (Multi-Tenant Ready)

### Config Registry (Replaces Flat `.env` for Org Definitions)

```
infra/config/
├── network.yml              # Global: channel, orderer, fabric version
├── orgs/
│   ├── org1.yml             # Per-org: ports, domains, peer, CA
│   ├── org2.yml
│   ├── org3.yml
│   └── org4.yml             # Adding an org = adding a file
└── environments/
    ├── dev.yml              # Dev overrides
    ├── staging.yml
    └── production.yml
```

### Per-Org Config (`orgs/org1.yml`)

```yaml
name: Org1MSP
role: manufacturer
domain: org1.example.com

peer:
  host: 10.0.0.2
  port: 8051
  couchdb_port: 5984

ca:
  port: 8054
  admin_id: ca-admin
  # secret from vault, not in file

api:
  port: 8080

postgres:
  host: 10.0.0.2
  port: 5436
  database: fabric_ca_org1

identities:
  admin: { id: admin }
  users:
    - { id: user1 }
  peer: { id: peer0 }
```

### Global Config (`network.yml`)

```yaml
fabric_version: "2.5.15"
ca_version: "1.5.5"
channel: mychannel
orderer:
  type: etcdraft
  host: 10.0.0.1
  port: 7050

endorsement_policy: majority  # auto-generates based on registered orgs
```

### Config Loader (Go CLI Tool + Library)

CLI tool (`cmd/configloader/`) wrapping a Go library. Reads all org YAMLs, merges with environment overrides, validates, and generates:
- Docker compose files per org
- `configtx.yaml` with all registered orgs
- Connection profiles per org
- Endorsement policy (auto-computed from registered orgs)

### Secrets

```yaml
# org1.yml — NO secrets in config files
ca:
  admin_secret: ${VAULT:fabric/org1/ca-admin-secret}
```

- Dev: `.env` file with defaults
- Production: OpenBao or Docker secrets

### Adding a New Org

1. Create `orgs/org4.yml`
2. Run `make add-org ORG=org4`
3. Script: reads config -> generates compose -> provisions CA -> enrolls identities -> joins channel -> updates endorsement policy

---

## 10. Testing Strategy

### 4 Levels

**Unit tests (every PR, <2 min):**

| Package | What's Tested | Tool |
|---------|--------------|------|
| Chaincode contracts | Business logic, validation, state transitions | `go test` + mock stub |
| Chaincode models | Serialization, key generation, status constants | `go test` |
| API handlers | Request parsing, response formatting | `go test` + `httptest` |
| API services | Business logic, error mapping | `go test` + mock Gateway |
| API middleware | JWT, rate limiting, CORS | `go test` + `httptest` |
| Config loader | YAML parsing, validation, generation | `go test` |
| Frontend | Components, stores, API client | `vitest` + `@testing-library/svelte` |

Target: 80%+ coverage on chaincode + API services.

**Integration tests (every PR, <5 min):**

- Chaincode against real CouchDB (Docker)
- API against real Fabric Gateway + test peer
- Config loader generates real compose files, validated with `docker compose config`

**E2E tests (on merge to main, ~10 min):**

Full supply chain flow:
1. Create Product (Org1)
2. Create Shipment (Org1)
3. Accept Custody (Org2)
4. Forward to Org3
5. Sell at POS (Org3)
6. Verify Provenance
7. Issue Recall
8. Check affected products

Playwright for frontend. Script-based for API-only flow.

**Contract tests (chaincode-specific):**

- State machine transitions (valid and invalid)
- Access control (org-scoped permissions)
- Endorsement requirements

### Test File Convention

```
contracts/product.go        -> contracts/product_test.go
internal/handler/product.go -> internal/handler/product_test.go
internal/service/fabric.go  -> internal/service/fabric_test.go
```

---

## Implementation Priority

Recommended order (each phase is independently shippable):

1. **Phase 1: Restructure + Config** — New project structure, config-per-org, split chaincode and API. No new features, just reorganization.
2. **Phase 2: Security** — Input validation, auth hardening, secrets management, audit logging.
3. **Phase 3: Testing** — Unit tests for chaincode + API, integration test setup, CI pipeline.
4. **Phase 4: Frontend** — New SvelteKit app with role-based views, httpOnly auth, API proxy.
5. **Phase 5: Observability** — Prometheus + Loki + Grafana, dashboards, alerting.
6. **Phase 6: Deployment** — Dockerfiles, compose split, Makefile, Caddy proxy, deploy workflow.
7. **Phase 7: Multi-Tenant Foundation** — Config loader generates artifacts, `make add-org` automation.

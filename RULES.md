# RULES.md — HLF Demo Project

## Project Overview

Hyperledger Fabric 2.4.9 blockchain network with:
- 1 Orderer (etcdraft)
- 2 Peer Orgs (Org1, Org2)
- 3 Fabric CAs (orderer + per org)
- Go REST API (Gin) as client layer
- Go chaincode (basic asset management)
- Docker Compose for all services

---

## Directory Layout

```
hlf-demo/
├── config/                  # core.yaml + generated channel artifacts (.block, .tx)
├── fabric-binaries/         # Downloaded peer/orderer/fabric-ca-client binaries
├── fabric-network/
│   ├── chaincode/basic/     # Go chaincode
│   ├── client/              # Go REST API (Gin + Fabric Gateway)
│   ├── config/              # configtx.yaml, connection profiles
│   └── scripts/
│       ├── deployment/      # Numbered scripts 001–009, 999
│       └── helpers/         # fabric-env.sh, download-fabric-binaries.sh
├── organizations/           # Generated crypto (MSP, TLS certs, keys) — gitignored
├── docker-compose/          # Generated compose files — gitignored
├── backups/                 # Timestamped network state backups — gitignored
├── copy-script/             # rsync scripts for multi-machine sync
└── docs/                    # Project documentation
```

---

## Naming Conventions

### Network Domains
```
<component>.<org>.example.com
peer0.org1.example.com
ca.org2.example.com
orderer1.orderer.example.com
```

### MSP IDs
```
OrdererMSP, Org1MSP, Org2MSP
```

### Identities
```
admin.<org>.example.com
user1.<org>.example.com
peer0.<org>.example.com
bootstrap-admin.<identity>
```

### Environment Variables
- UPPERCASE_UNDERSCORE
- Prefixes: `DEPLOY_`, `POSTGRES_`, `CA_`, `PEER_`, `COUCHDB_`, `ORDERER_`, `ORG1_`, `ORG2_`

### Scripts
- Format: `NNN-<description>.sh` (001–009 sequential, 999 = teardown)
- Never reuse or skip numbers

### Chaincode Functions
- CamelCase: `CreateAsset`, `ReadAsset`, `TransferAsset`
- Struct fields: Go convention (exported PascalCase with JSON camelCase tags)

---

## Port Assignments

| Service | Port |
|---------|------|
| Orderer | 7050 |
| CA Orderer | 7054 |
| CA Org1 | 8054 |
| CA Org2 | 9054 |
| Peer Org1 | 8051 |
| Peer Org2 | 9051 |
| CouchDB Org1 | 5984 |
| CouchDB Org2 | 7984 |
| PostgreSQL Orderer CA | 5432 |
| PostgreSQL Org1 CA | 5433 |
| PostgreSQL Org2 CA | 5434 |
| REST API | 8080 |

Do not change port assignments without updating `.env.example` and all Docker Compose templates.

---

## Configuration Rules

1. **All config lives in `.env`** — never hardcode values in scripts or Go code
2. **`.env` is gitignored** — always keep `.env.example` in sync when adding new vars
3. **`.env.example` is the source of truth** for all configurable parameters
4. Connection profile (`connection-profile.yaml`) embeds TLS certs as PEM — regenerate after cert rotation
5. Deployment flags (`DEPLOY_ORDERER`, `DEPLOY_ORG1`, `DEPLOY_ORG2`) control which components start — respect them in all scripts

---

## Script Rules

1. Scripts run **sequentially** in numbered order (001 → 002 → ... → 009)
2. Each script is **idempotent where possible** — check existence before creating
3. Source `fabric-env.sh` at top of every script for binary paths and env
4. Scripts respect deployment flags — skip component if its `DEPLOY_*` flag is false
5. **999-teardown.sh** removes everything — never run in production without backup
6. Use `set -e` — fail fast on errors
7. Log meaningful progress messages to stdout
8. Chaincode install: handle "already successfully installed" exit code gracefully — skip with warning, still query package ID

---

## Chaincode Rules

1. Language: **Go only**
2. Location: `fabric-network/chaincode/<name>/`
3. Chaincode name matches directory name (currently `basic`)
4. Version bumps require re-approval + commit on both orgs — update `CHAINCODE_VERSION` in `.env`
5. Endorsement policy: `AND('Org1MSP.member','Org2MSP.member')` — both orgs must sign
6. All chaincode functions must return typed errors, never panic
7. Use `ctx.GetStub().GetStateByRange()` for range queries, not full table scans
8. Asset existence check before create/update/delete

---

## REST API Rules

1. Base path: `/api/v1/`
2. Route groups: `assets`, `channels`, `chaincodes`, `transactions`, `network`
3. All responses: JSON with consistent error structure `{"error": "message"}`
4. Health endpoint: `GET /health` — queries ledger via `GetAllAssets`; returns 503 if Fabric unreachable
5. No business logic in handlers — delegate to `fabric/connector.go`
6. Fabric Gateway connection initialized once at startup, reused across requests
7. CORS: set `CORS_ALLOWED_ORIGIN` env var — never use wildcard `*` in production
8. Default port 8080, configurable via `CLIENT_PORT` env var
9. All `/api/v1/` routes require JWT Bearer token — set `JWT_SECRET` env var (fail 500 if unset)
10. Validate all asset IDs: alphanumeric + `_-`, max 64 chars, before submitting to chaincode
11. Return specific HTTP codes: 400 (validation), 401 (auth), 404 (not found), 409 (conflict), 503 (ledger down)
12. Log full error server-side (`log.Printf("ERROR ...")`) — return generic message to client

---

## Crypto / Security Rules

1. **TLS everywhere** — all peer, orderer, CA communication uses TLS
2. Crypto materials live in `organizations/` — gitignored, never commit
3. Regenerate all certs through CA enrollment scripts (003-setup-ca.sh) — never create manually
4. User identities stored in `fabric-network/client/crypto/wallet/`
5. Rotate certs by re-running 002 + 003 + updating connection profile
6. CA admin credentials in `.env` — treat as secrets, never log
7. `JWT_SECRET` must be set to a strong random value in production — never use the dev default
8. `CORS_ALLOWED_ORIGIN` must list only trusted origins — never `*`

---

## Docker Rules

1. All services defined in generated Docker Compose files under `docker-compose/`
2. Resource limits (CPU/memory) set per service via env vars — always cap them
3. CouchDB used as state DB (not LevelDB) — required for rich queries
4. PostgreSQL used as CA backend (not SQLite) — required for production-grade CA
5. Container names follow domain convention: `peer0.org1.example.com`
6. All containers on same Docker network for single-machine; use external host vars for multi-machine

---

## Multi-Machine Deployment

- Use `*_EXTERNAL_HOST` env vars to point components at remote machines
- `copy-script/` rsync scripts sync crypto materials between machines
- Run only relevant org scripts on each machine (controlled by `DEPLOY_*` flags)
- Ensure firewall opens required ports before deployment
- For local single-machine: set all `*_EXTERNAL_HOST` vars to `localhost` and update `/etc/hosts` for peer/orderer domains
- After `/etc/hosts` changes, restart peer containers to flush Docker's per-container DNS cache

---

## Backup Rules

1. Backup before any network change: `backups/backup-<YYYYMMDD-HHMMSS>/`
2. Backup contains: `organizations/`, `config/channel-artifacts/`, `docker-compose/`
3. Retention: configurable via `BACKUP_RETENTION_DAYS` in `.env`
4. Never delete backups without verifying network is healthy

---

## Git Rules

1. Never commit: `organizations/`, `docker-compose/`, `config/channel-artifacts/`, `*.block`, `*.tx`, `*.tar.gz`, `.env`
2. Always commit: `.env.example` changes when adding new env vars
3. Commit messages: imperative, lowercase, short (`fix peer TLS path`, `add org2 CouchDB config`)
4. No WIP commits to main — branch for experiments

---

## Technology Versions (Pinned)

| Component | Version |
|-----------|---------|
| Hyperledger Fabric | 2.5.15 |
| Fabric CA | 1.5.5 |
| Fabric Gateway (Go SDK) | v1.5.1 |
| Go (chaincode) | 1.23 |
| Go (client) | 1.25 |
| Gin | v1.12.0 |
| gRPC | v1.67.3 |
| JWT | github.com/golang-jwt/jwt/v5 v5.3.1 |
| PostgreSQL | 14 (pgvector image) |

Update versions only intentionally — test full network after any version change.

---

## Channel Configuration

- Channel name: `mychannel` (hardcoded in scripts and connection profile)
- Profile: `TwoOrgsChannel`
- Genesis profile: `TwoOrgsOrdererGenesis`
- Changing channel name requires updating scripts, connection profile, and env vars

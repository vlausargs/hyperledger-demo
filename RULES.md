# RULES.md — HLF Demo Project

## Project Overview

Hyperledger Fabric 2.5.15 blockchain network with:
- 1 Orderer (etcdraft)
- 3 Peer Orgs (Org1MSP = Manufacturer, Org2MSP = Distributor, Org3MSP = Retailer)
- 4 Fabric CAs (orderer + one per org)
- PostgreSQL as CA backend (one DB per CA, schema-isolated)
- CouchDB as peer state DB (one per peer)
- Go supply chain chaincode (`basic`)
- Go REST API (Gin) — one client process per org (ports 8080 / 8081 / 8082)
- Docker Compose for all services

---

## Directory Layout

```
hlf-demo/
├── config/                  # core.yaml + generated channel artifacts (.block, .tx)
├── fabric-binaries/         # Downloaded peer/orderer/fabric-ca-client binaries
├── fabric-network/
│   ├── chaincode/basic/     # Go supply chain chaincode
│   │   └── META-INF/statedb/couchdb/indexes/  # CouchDB index definitions
│   ├── client/              # Go REST API (Gin + Fabric Gateway)
│   │   ├── crypto-Org1/     # Org1 TLS certs + user key/cert + connection profile
│   │   ├── crypto-Org2/     # Org2 (same structure)
│   │   ├── crypto-Org3/     # Org3 (same structure)
│   │   ├── wallet-Org1/     # Org1 identity wallet
│   │   ├── wallet-Org2/
│   │   └── wallet-Org3/
│   ├── config/              # configtx.yaml, connection profiles
│   └── scripts/
│       ├── deployment/      # Numbered scripts 001–009, 999
│       └── helpers/         # fabric-env.sh, download-fabric-binaries.sh
├── organizations/           # Generated crypto (MSP, TLS certs, keys) — gitignored
├── fabric-network/organizations/ # Same — gitignored (see .gitignore)
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
OrdererMSP, Org1MSP, Org2MSP, Org3MSP
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
- Prefixes: `DEPLOY_`, `POSTGRES_`, `CA_`, `PEER_`, `COUCHDB_`, `ORDERER_`, `ORG1_`, `ORG2_`, `ORG3_`

### Scripts
- Format: `NNN-<description>.sh` (001–009 sequential, 999 = teardown)
- Never reuse or skip numbers

### Chaincode Functions
- CamelCase: `CreateProduct`, `ReadProduct`, `CreateShipment`
- Struct fields: Go convention (exported PascalCase with JSON camelCase tags)

---

## Port Assignments

| Service | Port |
|---------|------|
| Orderer | 7050 |
| CA Orderer | 7054 |
| CA Org1 | 8054 |
| CA Org2 | 9054 |
| CA Org3 | 10054 |
| Peer Org1 | 8051 |
| Peer Org2 | 9051 |
| Peer Org3 | 10051 |
| CouchDB Org1 | 5984 |
| CouchDB Org2 | 7984 |
| CouchDB Org3 | 9984 |
| PostgreSQL Orderer CA (local) | 5435 |
| PostgreSQL Org1 CA (local) | 5436 |
| PostgreSQL Org2 CA (local) | 5437 |
| PostgreSQL Org3 CA (local) | 5438 |
| REST API Org1 (Manufacturer) | 8080 |
| REST API Org2 (Distributor) | 8081 |
| REST API Org3 (Retailer) | 8082 |

When `DEPLOY_POSTGRES=false` all CAs connect to external postgres at `POSTGRES_*_EXTERNAL_HOST:5432`.

Do not change port assignments without updating `.env.example` and all Docker Compose templates.

---

## Configuration Rules

1. **All config lives in `.env`** — never hardcode values in scripts or Go code
2. **`.env` is gitignored** — always keep `.env.example` in sync when adding new vars
3. **`.env.example` is the source of truth** for all configurable parameters
4. Connection profile (`connection-profile.yaml`) embeds TLS certs as PEM — regenerate after cert rotation
5. Deployment flags control which components start — respect them in all scripts:
   - `DEPLOY_ORDERER`, `DEPLOY_ORG1`, `DEPLOY_ORG2`, `DEPLOY_ORG3`
   - `DEPLOY_POSTGRES` — `false` skips local postgres containers; CAs use `POSTGRES_*_EXTERNAL_HOST`
   - `DEPLOY_COUCHDB` — `false` skips local CouchDB containers; peers use `COUCHDB_ORGn_HOST`

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
9. **Teardown grep pattern** matches `postgres-ord|postgres-org` (not bare `postgres`) — prevents killing external postgres containers not managed by these scripts
10. **002-deploy-ca.sh**: when `DEPLOY_POSTGRES=true`, CA datasource uses `localhost` (not container name) because both CA and postgres share `network_mode: host`; uses `--force-recreate` on compose up so containers reload config on re-run

---

## PostgreSQL / CA Database Rules

1. Each CA uses its own database: `fabric_ca_orderer`, `fabric_ca_org1`, `fabric_ca_org2`, `fabric_ca_org3`
2. Tables are created in a schema matching the db name (not `public`) via `search_path=<db_name>` in datasource
3. Schema is created by `002-deploy-ca.sh` (`ensure_pg_schema`) before CA starts
4. Datasource must include `sslmode=disable` — `sslmode=prefer` causes silent connection failures with lib/pq when postgres has no SSL configured
5. Password special characters must be quoted: `password='p@ss!'` in datasource string
6. When `DEPLOY_POSTGRES=true`: local containers bind unique ports (5435–5438), CA connects via `localhost`
7. When `DEPLOY_POSTGRES=false`: CA connects to `POSTGRES_*_EXTERNAL_HOST:5432`; schema and user must pre-exist

---

## Chaincode Rules

1. Language: **Go only**
2. Location: `fabric-network/chaincode/<name>/`
3. Chaincode name matches directory name (currently `basic`)
4. Version bumps require re-approval + commit on all 3 orgs — update `CHAINCODE_VERSION` in `.env`
5. Endorsement policy: `AND('Org1MSP.member','Org2MSP.member','Org3MSP.member')` — all 3 orgs must sign
6. All chaincode functions must return typed errors, never panic
7. No `DelState` — supply chain records are permanent audit trail
8. Asset existence check before create/update
9. Ledger key prefixes: `PROD~`, `SHIP~`, `CUSTODY~`, `EVENT~`, `RECALL~`
10. CouchDB indexes defined in `META-INF/statedb/couchdb/indexes/` — required for rich queries on `docType`, `status`, `batchId`, `currentOwnerMSP`

### Domain Models
| Type | Key |
|------|-----|
| Product | `PROD~{id}` |
| Shipment | `SHIP~{id}` |
| CustodyRecord | `CUSTODY~{shipmentId}~{seq:04d}` |
| SupplyChainEvent | `EVENT~{targetId}~{txTimestamp}~{id}` |
| RecallNotice | `RECALL~{id}` |

---

## REST API Rules

1. Base path: `/api/v1/`
2. Route groups: `products`, `shipments`, `events`, `recalls`, `channels`, `chaincodes`, `network`, `auth`
3. All responses: JSON with consistent error structure `{"error": "message"}`
4. Health endpoint: `GET /health` — queries ledger via `GetAllProducts`; returns 503 if Fabric unreachable
5. No business logic in handlers — delegate to `fabric/connector.go`
6. Fabric Gateway connection initialized once at startup, reused across requests
7. CORS: set `CORS_ALLOWED_ORIGIN` env var — never use wildcard `*` in production
8. **Three client processes**, one per org, on separate ports:
   - Org1 (Manufacturer) → 8080
   - Org2 (Distributor) → 8081
   - Org3 (Retailer) → 8082
9. Each client uses its own `crypto-OrgN/` dir, `wallet-OrgN/` dir, and connection profile
10. All `/api/v1/` routes require JWT Bearer token — set `JWT_SECRET` env var
11. Validate all IDs: alphanumeric + `_-`, max 64 chars, before submitting to chaincode
12. Return specific HTTP codes: 400 (validation), 401 (auth), 404 (not found), 409 (conflict), 503 (ledger down)
13. Log full error server-side — return generic message to client

---

## Crypto / Security Rules

1. **TLS everywhere** — all peer, orderer, CA communication uses TLS
2. Crypto materials live in `organizations/` — gitignored, never commit
3. Regenerate all certs through CA enrollment scripts (003-setup-ca.sh) — never create manually
4. User identities stored in `fabric-network/client/wallet-OrgN/`
5. Rotate certs by re-running 002 + 003 + updating connection profile
6. CA admin credentials in `.env` — treat as secrets, never log
7. `JWT_SECRET` must be set to a strong random value in production — never use the dev default
8. `CORS_ALLOWED_ORIGIN` must list only trusted origins — never `*`

---

## Docker Rules

1. All services defined in generated Docker Compose files under `docker-compose/`
2. Resource limits (CPU/memory) set per service via env vars — always cap them
3. CouchDB used as state DB (not LevelDB) — required for rich queries; skip local containers with `DEPLOY_COUCHDB=false`
4. PostgreSQL used as CA backend (not SQLite) — required for production-grade CA; skip local containers with `DEPLOY_POSTGRES=false`
5. Container names follow domain convention: `peer0.org1.example.com`, `postgres-org1`, `ca-org1`
6. All containers use `network_mode: host` — do not add bridge network entries
7. When `DEPLOY_POSTGRES=true`, each postgres container must bind a unique host port (5435–5438)
8. Teardown only removes containers matching fabric-managed name patterns — external containers must use distinct names

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

1. Never commit: `organizations/`, `fabric-network/organizations/`, `docker-compose/`, `config/channel-artifacts/`, `*.block`, `*.tx`, `*.tar.gz`, `.env`
2. Always commit: `.env.example` changes when adding new env vars
3. Commit messages: imperative, lowercase, short (`fix peer TLS path`, `add org3 CouchDB config`)
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

- Channel name: `mychannel`
- Profile: `ThreeOrgsChannel`
- Genesis profile: `ThreeOrgsOrdererGenesis`
- Changing channel name requires updating scripts, connection profile, and env vars

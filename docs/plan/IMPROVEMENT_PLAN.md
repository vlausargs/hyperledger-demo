# Improvement Plan

Priority order: Critical → High → Medium → Low. Each item has exact file + line, what to change, and how.

---

## Phase 1 — Critical Security

### 1.1 Add API Authentication (JWT)
**File:** `fabric-network/client/main.go`, `fabric-network/client/rest/`

No auth today. Every endpoint is public.

**Plan:**
1. Add `github.com/golang-jwt/jwt/v5` to `go.mod`
2. Create `fabric-network/client/middleware/auth.go`:
   ```go
   func JWTMiddleware(secret string) gin.HandlerFunc {
       return func(c *gin.Context) {
           tokenStr := c.GetHeader("Authorization")
           // strip "Bearer " prefix, parse JWT, validate claims
           // c.AbortWithStatus(401) on failure
           c.Next()
       }
   }
   ```
3. In `main.go`, apply middleware to `v1` group only (not `/health`):
   ```go
   v1 := router.Group("/api/v1")
   v1.Use(middleware.JWTMiddleware(os.Getenv("JWT_SECRET")))
   ```
4. Add `JWT_SECRET` to `.env.example`
5. Document token issuance process (out-of-band for now — static token or separate auth service)

**Scope:** New file + ~10 lines in `main.go`

---

### 1.2 Fix CORS — Remove Wildcard + Credentials
**File:** `fabric-network/client/main.go:197-198`

Current:
```go
c.Header("Access-Control-Allow-Origin", "*")
c.Header("Access-Control-Allow-Credentials", "true")  // INVALID with wildcard
```

**Plan:**
```go
allowedOrigin := getEnv("CORS_ALLOWED_ORIGIN", "http://localhost:3000")
c.Header("Access-Control-Allow-Origin", allowedOrigin)
c.Header("Access-Control-Allow-Credentials", "true")
```

Add `CORS_ALLOWED_ORIGIN` to `.env.example`. For production set to actual frontend domain.

**Scope:** 3 lines changed in `corsMiddleware()`

---

### 1.3 Add Input Validation for Asset ID
**File:** `fabric-network/client/rest/handlers.go`

`c.Param("id")` used raw at lines 102, 159, 193, 213, 242, 325, 361.

**Plan:**
1. Create `fabric-network/client/rest/validate.go`:
   ```go
   var validAssetID = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

   func validateAssetID(id string) error {
       if !validAssetID.MatchString(id) {
           return fmt.Errorf("invalid asset ID: must be alphanumeric, 1-64 chars")
       }
       return nil
   }
   ```
2. Call at top of each handler that takes `id`:
   ```go
   id := c.Param("id")
   if err := validateAssetID(id); err != nil {
       c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
       return
   }
   ```

**Scope:** New 10-line file + ~6 handler updates

---

### 1.4 Sanitize Error Responses — Don't Leak Internals
**File:** `fabric-network/client/rest/handlers.go` — all error returns

Current pattern everywhere:
```go
"error": fmt.Sprintf("Failed to get all assets: %v", err)
```

**Plan:**
1. Log full error server-side
2. Return generic message to client
3. For known Fabric errors (asset not found), return specific user-friendly message

```go
// In handlers
if err != nil {
    log.Printf("ERROR GetAllAssets: %v", err)
    // Check for known error types
    if strings.Contains(err.Error(), "does not exist") {
        c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
        return
    }
    c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
    return
}
```

**Scope:** ~20 error-return sites in `handlers.go`

---

### 1.5 Add Chaincode Access Control
**File:** `fabric-network/chaincode/basic/chaincode.go`

Any user can create/modify/delete any asset regardless of ownership.

**Plan:**
Add caller identity check to write functions:

```go
func (s *SmartContract) getCallerMSPID(ctx contractapi.TransactionContextInterface) (string, error) {
    id, err := ctx.GetClientIdentity().GetMSPID()
    if err != nil {
        return "", fmt.Errorf("failed to get client MSPID: %w", err)
    }
    return id, nil
}
```

For `DeleteAsset` and `TransferAsset` — verify caller owns the asset or is admin:
```go
callerMSP, err := s.getCallerMSPID(ctx)
if err != nil {
    return err
}
// Only owner's org can transfer/delete
// Store ownerMSP in Asset struct, check callerMSP == asset.OwnerMSP
```

**Requires:** Add `OwnerMSP string` field to `Asset` struct + update `CreateAsset` to record caller's MSPID.

**Scope:** Asset struct + 3 write functions + new helper

---

## Phase 2 — High: Reliability

### 2.1 Fix gRPC Dial — Add Timeout
**File:** `fabric-network/client/fabric/connector.go:208-217`

Current:
```go
grpcConnection, err := grpc.Dial(
    peerEndpoint,
    grpc.WithTransportCredentials(grpcCredentials),
    grpc.WithBlock(),   // ← hangs forever if peer down
    ...
)
```

**Plan:**
```go
dialCtx, dialCancel := context.WithTimeout(context.Background(), 30*time.Second)
defer dialCancel()

grpcConnection, err := grpc.DialContext(
    dialCtx,
    peerEndpoint,
    grpc.WithTransportCredentials(grpcCredentials),
    grpc.WithBlock(),
    grpc.WithKeepaliveParams(keepalive.ClientParameters{...}),
)
```

**Scope:** ~5 lines in `connect()`

---

### 2.2 Use SubmitAsync + Check Commit Status
**File:** `fabric-network/client/fabric/connector.go:334-345`
**File:** `fabric-network/client/rest/handlers.go` — write handlers

Current `SubmitTransaction` returns after orderer receives TX. Commit can still fail.

**Plan:**
Update `SubmitTransaction` in connector to check commit status:
```go
func (gw *Gateway) SubmitTransaction(function string, args ...string) ([]byte, error) {
    result, commit, err := gw.contract.SubmitAsync(function, client.WithArguments(args...))
    if err != nil {
        return nil, fmt.Errorf("failed to submit transaction: %w", err)
    }

    status, err := commit.Status()
    if err != nil {
        return nil, fmt.Errorf("failed to get commit status: %w", err)
    }
    if !status.Successful {
        return nil, fmt.Errorf("transaction committed with failure status: %v", status.Code)
    }

    return result, nil
}
```

**Scope:** Replace `SubmitTransaction` implementation in `connector.go`

---

### 2.3 Real Health Check
**File:** `fabric-network/client/rest/handlers.go:62-68`

Current:
```go
func HealthCheck(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"status": "healthy", ...})
}
```

**Plan:**
```go
func HealthCheck(gw FabricGateway) gin.HandlerFunc {
    return func(c *gin.Context) {
        _, err := gw.EvaluateTransaction("GetAllAssets")
        if err != nil {
            c.JSON(http.StatusServiceUnavailable, gin.H{
                "status": "unhealthy",
                "error":  "ledger unreachable",
            })
            return
        }
        c.JSON(http.StatusOK, gin.H{"status": "healthy"})
    }
}
```

Update `main.go:69` to pass gateway:
```go
router.GET("/health", rest.HealthCheck(fabricGateway))
```

**Scope:** `HealthCheck` sig change + `main.go` call site

---

### 2.4 Fix Placeholder Endpoints
**File:** `fabric-network/client/rest/handlers.go:306-451`

`GetPeers`, `GetOrganizations`, `GetTransactions`, `GetTransaction`, `GetChannels` return fake hardcoded data.

**Plan:**
- `GetChannels` / `GetChannelInfo` / `GetChaincodes` / `GetChaincodeInfo` — these can return real data from connection profile (already loaded in gateway). Pull from `gw.GetConnectionProfile()`.
- `GetPeers` — return peers from connection profile instead of hardcoded strings
- `GetOrganizations` — return orgs from connection profile
- `GetTransactions` / `GetTransaction` — Fabric Gateway SDK does not expose raw TX query. Return `501 Not Implemented` with honest message.

**Scope:** Rewrite 6 handler functions

---

### 2.5 Fix go.mod Version
**File:** `fabric-network/client/go.mod:3`

`go 1.25.0` does not exist (Go is at 1.23.x as of 2025).

**Plan:**
```
go 1.23
```

**Scope:** 1 line

---

## Phase 3 — Medium: Code Quality

### 3.1 Remove Dead Code
**File:** `fabric-network/client/fabric/connector.go:289-294`

```go
mspConfigBytes, err := os.ReadFile(mspConfigPath)
if err == nil {
    _ = mspConfigBytes // reads file and discards
}
```

**Plan:** Delete lines 289-294 entirely.

---

### 3.2 Add GetAllAssets Pagination
**File:** `fabric-network/chaincode/basic/chaincode.go:166`
**File:** `fabric-network/client/rest/handlers.go:71`

Open-ended range query returns entire ledger.

**Plan — Chaincode:**
```go
func (s *SmartContract) GetAllAssets(ctx contractapi.TransactionContextInterface, pageSize int32, bookmark string) ([]*Asset, string, error) {
    resultsIterator, metadata, err := ctx.GetStub().GetStateByRangeWithPagination("", "", pageSize, bookmark)
    // ...
    return assets, metadata.Bookmark, nil
}
```

**Plan — Handler:**
```go
pageSize := c.DefaultQuery("pageSize", "20")
bookmark := c.DefaultQuery("bookmark", "")
// parse pageSize, call EvaluateTransaction("GetAllAssets", pageSize, bookmark)
// return bookmark in response for client to paginate
```

**Scope:** Chaincode function signature change + handler update

---

### 3.3 Standardize Error Wrapping
**File:** All Go files

Mix of `%w` (wrappable) and `%v` (not wrappable) in `fmt.Errorf`.

**Plan:** Change all `fmt.Errorf("...: %v", err)` → `fmt.Errorf("...: %w", err)` where err is being wrapped and propagated up. Allows `errors.Is` / `errors.As` to work through chain.

**Scope:** ~15 sites across `chaincode.go`, `connector.go`, `handlers.go`

---

### 3.4 Replace Fixed Script Sleeps with Polling
**Files:** `fabric-network/scripts/deployment/002-deploy-ca.sh`, `003-setup-ca.sh`

Lines like `sleep 15` and `sleep 10` break on slow/fast machines.

**Plan:** Replace with poll-until-ready pattern:
```bash
wait_for_container() {
    local container=$1
    local max_wait=${2:-60}
    local elapsed=0
    until docker inspect "$container" --format='{{.State.Running}}' 2>/dev/null | grep -q true; do
        sleep 2
        elapsed=$((elapsed + 2))
        if [ $elapsed -ge $max_wait ]; then
            echo "ERROR: $container did not start within ${max_wait}s"
            exit 1
        fi
    done
}
```

**Scope:** Helper function in `fabric-env.sh` + replace sleep calls in deployment scripts

---

### 3.5 Fix CA Database SSL
**File:** `fabric-network/scripts/deployment/002-deploy-ca.sh`

```bash
datasource: ... sslmode=disable
```

**Plan:** Change to `sslmode=require` and mount Postgres TLS certs into CA container via Docker Compose volume. Or at minimum `sslmode=prefer` which upgrades if available.

**Scope:** 1 line in datasource template string

---

## Phase 4 — Low: Observability

### 4.1 Add Correlation IDs
**File:** `fabric-network/client/main.go`

**Plan:**
```go
func correlationIDMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        id := c.GetHeader("X-Correlation-ID")
        if id == "" {
            id = generateUUID()
        }
        c.Set("correlationID", id)
        c.Header("X-Correlation-ID", id)
        c.Next()
    }
}
```
Apply before `requestLogger()`. Include `correlationID` in log output.

---

### 4.2 Structured Logging
**File:** `fabric-network/client/` all files

Current: `log.Printf` plain strings.

**Plan:** Use stdlib `log/slog` (Go 1.21+, already in range):
```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
logger.Info("transaction evaluated", "function", function, "duration", elapsed)
```

No new dependency. Replace `log.Logger` in `Gateway` struct with `*slog.Logger`.

---

## Implementation Order

| Phase | Item | Effort | Risk |
|-------|------|--------|------|
| 1 | 1.2 Fix CORS | 15 min | None |
| 1 | 1.3 Input validation | 30 min | None |
| 1 | 1.4 Sanitize errors | 1h | None |
| 2 | 2.5 Fix go.mod | 5 min | None |
| 2 | 3.1 Remove dead code | 5 min | None |
| 2 | 2.1 gRPC dial timeout | 15 min | Low |
| 2 | 2.2 SubmitAsync commit check | 1h | Medium — changes TX behavior |
| 2 | 2.3 Real health check | 30 min | None |
| 2 | 2.4 Fix placeholder endpoints | 2h | None |
| 1 | 1.1 Add JWT auth | 2h | High — breaking change for clients |
| 1 | 1.5 Chaincode access control | 3h | High — requires redeploy + version bump |
| 3 | 3.2 Pagination | 2h | Medium — chaincode redeploy |
| 3 | 3.3 Error wrapping | 1h | None |
| 3 | 3.4 Script polling | 2h | None |
| 3 | 3.5 CA SSL | 30 min | Medium — needs cert setup |
| 4 | 4.1 Correlation IDs | 30 min | None |
| 4 | 4.2 Structured logging | 2h | None |

**Total estimated effort:** ~18-20 hours

---

## Files Changed Summary

| File | Changes |
|------|---------|
| `client/main.go` | CORS fix, JWT middleware wiring, health check fix |
| `client/rest/handlers.go` | Input validation, error sanitization, health check sig, placeholder fixes |
| `client/rest/validate.go` | New — asset ID validation |
| `client/middleware/auth.go` | New — JWT middleware |
| `client/fabric/connector.go` | gRPC timeout, SubmitAsync, remove dead code, structured logging |
| `client/go.mod` | Fix go version |
| `chaincode/basic/chaincode.go` | Access control, pagination, error wrapping |
| `scripts/deployment/002-deploy-ca.sh` | SSL fix, replace sleeps |
| `scripts/deployment/003-setup-ca.sh` | Replace sleeps |
| `scripts/helpers/fabric-env.sh` | Add poll helper function |
| `.env.example` | Add JWT_SECRET, CORS_ALLOWED_ORIGIN |

---

## What NOT Included (Out of Scope)

- **Secret management (Vault/KMS)** — environment variable approach is acceptable for demo; document secure production setup instead
- **Rate limiting** — adds dependency; document as follow-up
- **Full test suite** — separate effort, tracked separately
- **Fabric upgrade 2.4.9 → 2.5.x** — network-wide change, separate migration plan needed
- **CA TLS enablement** — requires cert chain setup, separate ops task

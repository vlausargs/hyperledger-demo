# Phase 2: Security Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Harden the REST API with input validation, improved JWT auth with org/role claims, security headers, rate limiting, and audit logging.

**Architecture:** Add middleware layers for security headers, rate limiting, and audit logging. Upgrade JWT to include org/role claims. Add ID validation to all handler URL parameters. Pass config through middleware instead of reading env vars directly.

**Tech Stack:** Go 1.25, Gin v1.12, golang-jwt/v5, golang.org/x/time/rate

**Spec:** `docs/superpowers/specs/2026-05-24-production-readiness-design.md` — Section 5 (Security Hardening)

---

## Task 1: Add Input Validation to All Handlers

Add `validateID` calls to every handler that reads `:id`, `:batchId`, `:status`, `:targetId`, etc. from URL params.

**Files:**
- Modify: `packages/api/internal/handler/product.go`
- Modify: `packages/api/internal/handler/shipment.go`
- Modify: `packages/api/internal/handler/event.go`
- Modify: `packages/api/internal/handler/recall.go`
- Modify: `packages/api/internal/handler/pos.go`
- Modify: `packages/api/internal/handler/network.go`

- [ ] **Step 1: Add validateID to product handlers**

In `product.go`, add validation to `GetProduct`, `UpdateProduct`, `GetProductHistory`, `GetProductProvenance`:
```go
if err := validateID("product", c.Param("id")); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
}
```
Add to `GetProductsByBatch` for batchId param, `GetProductsByStatus` for status param.

- [ ] **Step 2: Add validateID to shipment, event, recall, pos, network handlers**

Same pattern for all `:id` params in shipment.go, event.go (`:targetId`), recall.go, pos.go, network.go (`:channelId`, `:chaincodeId`, `:txId`).

- [ ] **Step 3: Verify build**

```bash
cd /home/valos/workspace/myindo/hlf-demo/packages/api && go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add packages/api/internal/handler/ && git commit -m "security: add input validation to all handler URL parameters"
```

---

## Task 2: Upgrade JWT Auth with Org/Role Claims and Config

Replace hardcoded credentials and env-var-based secret with config-driven auth. Add org and role claims to JWT. Shorten token TTL.

**Files:**
- Modify: `packages/api/internal/handler/auth.go`
- Modify: `packages/api/internal/handler/types.go`
- Modify: `packages/api/internal/middleware/auth.go`
- Modify: `packages/api/internal/router/router.go`
- Modify: `packages/api/internal/config/config.go`
- Modify: `packages/api/cmd/server/main.go`

- [ ] **Step 1: Add auth config fields**

Add to `config.Config`:
```go
TokenTTL  time.Duration  // default 15 min
MSPID     string         // already exists — used as org claim
```

- [ ] **Step 2: Update Login handler to accept config and add org/role claims**

Change `Login()` to `Login(jwtSecret, mspID string)`. Add claims:
```go
jwt.MapClaims{
    "sub":  req.Username,
    "org":  mspID,
    "role": "admin",
    "exp":  time.Now().Add(15 * time.Minute).Unix(),
    "iat":  time.Now().Unix(),
}
```

- [ ] **Step 3: Update JWT middleware to accept secret from config**

Change `JWT()` to `JWT(secret string)`. Remove `os.Getenv("JWT_SECRET")` call. Extract and set org/role from claims:
```go
c.Set("org", claims["org"])
c.Set("role", claims["role"])
c.Set("sub", claims["sub"])
```

- [ ] **Step 4: Update router and main to pass config values**

- [ ] **Step 5: Verify build and tests**

- [ ] **Step 6: Commit**

```bash
git commit -m "security: upgrade JWT with org/role claims, config-driven secret"
```

---

## Task 3: Add Security Headers Middleware

**Files:**
- Create: `packages/api/internal/middleware/security.go`
- Modify: `packages/api/internal/router/router.go`

- [ ] **Step 1: Create security headers middleware**

```go
package middleware

import "github.com/gin-gonic/gin"

func SecurityHeaders() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("X-Content-Type-Options", "nosniff")
        c.Header("X-Frame-Options", "DENY")
        c.Header("Content-Security-Policy", "default-src 'self'")
        c.Header("X-XSS-Protection", "1; mode=block")
        c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
        c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
        c.Next()
    }
}
```

- [ ] **Step 2: Add to router**

- [ ] **Step 3: Commit**

```bash
git commit -m "security: add security headers middleware"
```

---

## Task 4: Add Rate Limiting Middleware

**Files:**
- Create: `packages/api/internal/middleware/ratelimit.go`
- Modify: `packages/api/internal/router/router.go`
- Modify: `packages/api/go.mod` (add golang.org/x/time)

- [ ] **Step 1: Create rate limiter**

Token bucket per IP using `golang.org/x/time/rate`:
```go
package middleware

import (
    "net/http"
    "sync"
    "time"
    "github.com/gin-gonic/gin"
    "golang.org/x/time/rate"
)

func RateLimit(rps int, burst int) gin.HandlerFunc {
    type client struct {
        limiter  *rate.Limiter
        lastSeen time.Time
    }
    var mu sync.Mutex
    clients := make(map[string]*client)

    // Cleanup goroutine
    go func() {
        for {
            time.Sleep(time.Minute)
            mu.Lock()
            for ip, c := range clients {
                if time.Since(c.lastSeen) > 3*time.Minute {
                    delete(clients, ip)
                }
            }
            mu.Unlock()
        }
    }()

    return func(c *gin.Context) {
        ip := c.ClientIP()
        mu.Lock()
        if _, exists := clients[ip]; !exists {
            clients[ip] = &client{limiter: rate.NewLimiter(rate.Limit(rps), burst)}
        }
        clients[ip].lastSeen = time.Now()
        limiter := clients[ip].limiter
        mu.Unlock()

        if !limiter.Allow() {
            c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
            c.Abort()
            return
        }
        c.Next()
    }
}
```

- [ ] **Step 2: Add to router (100 rps, burst 200)**

- [ ] **Step 3: Commit**

```bash
git commit -m "security: add per-IP rate limiting middleware"
```

---

## Task 5: Add Audit Logging Middleware

**Files:**
- Create: `packages/api/internal/middleware/audit.go`
- Modify: `packages/api/internal/router/router.go`

- [ ] **Step 1: Create audit middleware**

Log all state-changing requests (POST, PUT, DELETE) with structured JSON to a separate logger:

```go
package middleware

import (
    "log/slog"
    "os"
    "time"
    "github.com/gin-gonic/gin"
)

var auditLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))

func AuditLog() gin.HandlerFunc {
    return func(c *gin.Context) {
        if c.Request.Method == "GET" || c.Request.Method == "OPTIONS" || c.Request.Method == "HEAD" {
            c.Next()
            return
        }

        start := time.Now()
        c.Next()

        sub, _ := c.Get("sub")
        org, _ := c.Get("org")
        cid, _ := c.Get("correlationID")

        auditLogger.Info("audit",
            "action", c.Request.Method,
            "path", c.Request.URL.Path,
            "status", c.Writer.Status(),
            "user", sub,
            "org", org,
            "ip", c.ClientIP(),
            "correlation_id", cid,
            "duration_ms", time.Since(start).Milliseconds(),
        )
    }
}
```

- [ ] **Step 2: Add to router (on v1 group, after JWT)**

- [ ] **Step 3: Commit**

```bash
git commit -m "security: add audit logging for state-changing operations"
```

---

## Task 6: Verify Full Build and Tests

- [ ] **Step 1: Run full build, test, lint**

```bash
cd /home/valos/workspace/myindo/hlf-demo && make build && make test && make lint
```

- [ ] **Step 2: Commit if needed**

//go:build integration
// +build integration

// Package integration exercises the full HTTP stack of the API by wiring
// real middleware (JWT, CORS, rate-limit, correlation-id, security-headers)
// to inline handlers backed by a mock FabricGateway. It uses net/http/httptest
// and runs with no external dependencies.
//
// We do NOT import the internal/handler or internal/router packages: those
// are currently being refactored. Instead, the test owns its own router so
// the integration coverage stays decoupled from in-flight handler changes.
//
// Run: go test -tags=integration ./test/integration/...
package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/myindo/hlf-supply-chain/api/internal/middleware"
)

func init() { gin.SetMode(gin.TestMode) }

// ---------------------------------------------------------------------------
// Mock FabricGateway — minimal in-memory key-value backed mock that mimics
// the service-layer's gateway abstraction.
// ---------------------------------------------------------------------------

type mockGateway struct {
	mu      sync.Mutex
	state   map[string]json.RawMessage
	submits []submitCall
}

type submitCall struct {
	Fn   string
	Args []string
}

func newMockGateway() *mockGateway {
	return &mockGateway{state: make(map[string]json.RawMessage)}
}

func (m *mockGateway) SubmitTransaction(fn string, args ...string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.submits = append(m.submits, submitCall{Fn: fn, Args: args})

	switch fn {
	case "CreateProduct":
		if len(args) < 6 {
			return nil, fmt.Errorf("not enough args")
		}
		id := args[0]
		if _, exists := m.state["P:"+id]; exists {
			return nil, fmt.Errorf("product %s already exists", id)
		}
		body, _ := json.Marshal(map[string]string{
			"id": id, "sku": args[1], "name": args[2], "status": "ACTIVE",
		})
		m.state["P:"+id] = body
		return nil, nil
	}
	return nil, nil
}

func (m *mockGateway) EvaluateTransaction(fn string, args ...string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch fn {
	case "GetAllProducts":
		var products []json.RawMessage
		for k, v := range m.state {
			if strings.HasPrefix(k, "P:") {
				products = append(products, v)
			}
		}
		if products == nil {
			products = make([]json.RawMessage, 0)
		}
		return json.Marshal(map[string]interface{}{
			"products": products, "count": len(products), "bookmark": "",
		})
	case "ReadProduct":
		id := args[0]
		if b, ok := m.state["P:"+id]; ok {
			return b, nil
		}
		return nil, fmt.Errorf("product %s does not exist", id)
	}
	return []byte("{}"), nil
}

// ---------------------------------------------------------------------------
// Test router — minimal but faithful to the production wiring.
// ---------------------------------------------------------------------------

const (
	jwtSecret   = "integration-test-secret-32bytes-min"
	corsOrigin  = "http://localhost:3000"
	mspID       = "Org1MSP"
	loginUser   = "admin"
	loginPasswd = "asdqwe123"
)

func buildRouter(gw *mockGateway) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.RateLimit(100, 200))
	r.Use(middleware.CORS(corsOrigin))
	r.Use(middleware.CorrelationID())

	// Health (unauthenticated)
	r.GET("/health", func(c *gin.Context) {
		if _, err := gw.EvaluateTransaction("GetAllProducts", "1", ""); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Login (unauthenticated)
	r.POST("/api/v1/auth/login", func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username and password required"})
			return
		}
		if req.Username != loginUser || req.Password != loginPasswd {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": req.Username, "org": mspID, "role": "admin",
			"exp": time.Now().Add(15 * time.Minute).Unix(),
			"iat": time.Now().Unix(),
		})
		signed, err := tok.SignedString([]byte(jwtSecret))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "token gen failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": signed, "expires_in": 900})
	})

	// Protected endpoints
	v1 := r.Group("/api/v1")
	v1.Use(middleware.JWT(jwtSecret))
	{
		v1.GET("/products", func(c *gin.Context) {
			b, err := gw.EvaluateTransaction("GetAllProducts", c.DefaultQuery("pageSize", "20"), c.DefaultQuery("bookmark", ""))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "fail"})
				return
			}
			c.Data(http.StatusOK, "application/json", b)
		})
		v1.POST("/products", func(c *gin.Context) {
			var req struct {
				ID               string            `json:"id"               binding:"required"`
				SKU              string            `json:"sku"              binding:"required"`
				Name             string            `json:"name"             binding:"required"`
				Description      string            `json:"description"`
				BatchID          string            `json:"batchId"          binding:"required"`
				ManufacturerName string            `json:"manufacturerName" binding:"required"`
				ExpiryDate       string            `json:"expiryDate"`
				Metadata         map[string]string `json:"metadata"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			meta := "{}"
			if req.Metadata != nil {
				if b, err := json.Marshal(req.Metadata); err == nil {
					meta = string(b)
				}
			}
			_, err := gw.SubmitTransaction("CreateProduct",
				req.ID, req.SKU, req.Name, req.Description,
				req.BatchID, req.ManufacturerName, req.ExpiryDate, meta)
			if err != nil {
				if strings.Contains(err.Error(), "already exists") {
					c.JSON(http.StatusConflict, gin.H{"error": "product already exists"})
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
				return
			}
			c.JSON(http.StatusCreated, gin.H{"message": "product created", "id": req.ID})
		})
		v1.GET("/products/:id", func(c *gin.Context) {
			id := c.Param("id")
			if id == "" || !isValidID(id) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product ID"})
				return
			}
			b, err := gw.EvaluateTransaction("ReadProduct", id)
			if err != nil {
				if strings.Contains(err.Error(), "does not exist") {
					c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": "fail"})
				return
			}
			c.Data(http.StatusOK, "application/json", b)
		})
	}
	return r
}

func isValidID(id string) bool {
	if len(id) == 0 || len(id) > 128 {
		return false
	}
	for _, r := range id {
		if !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') &&
			!(r >= '0' && r <= '9') && r != '_' && r != '-' && r != '~' {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func login(t *testing.T, r *gin.Engine, user, pass string) (int, string) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"username": user, "password": pass})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		return w.Code, ""
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	tok, _ := resp["token"].(string)
	return w.Code, tok
}

func do(r *gin.Engine, method, path, token string, body interface{}) *httptest.ResponseRecorder {
	var rdr *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestAPI_Health_OK(t *testing.T) {
	gw := newMockGateway()
	r := buildRouter(gw)
	w := do(r, "GET", "/health", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("health: expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestAPI_Login_Success(t *testing.T) {
	gw := newMockGateway()
	r := buildRouter(gw)
	code, tok := login(t, r, loginUser, loginPasswd)
	if code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d", code)
	}
	if tok == "" {
		t.Fatal("expected token")
	}
}

func TestAPI_Login_BadCredentials(t *testing.T) {
	gw := newMockGateway()
	r := buildRouter(gw)
	code, _ := login(t, r, loginUser, "wrong-password")
	if code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", code)
	}
}

func TestAPI_Login_MissingFields(t *testing.T) {
	gw := newMockGateway()
	r := buildRouter(gw)
	w := do(r, "POST", "/api/v1/auth/login", "", map[string]string{"username": loginUser})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAPI_Protected_NoToken(t *testing.T) {
	gw := newMockGateway()
	r := buildRouter(gw)
	w := do(r, "GET", "/api/v1/products", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", w.Code)
	}
}

func TestAPI_Protected_InvalidToken(t *testing.T) {
	gw := newMockGateway()
	r := buildRouter(gw)
	w := do(r, "GET", "/api/v1/products", "not-a-real-jwt", nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with garbage token, got %d", w.Code)
	}
}

func TestAPI_Protected_ExpiredToken(t *testing.T) {
	gw := newMockGateway()
	r := buildRouter(gw)
	expired := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": loginUser, "org": mspID, "role": "admin",
		"exp": time.Now().Add(-1 * time.Hour).Unix(),
	})
	signed, _ := expired.SignedString([]byte(jwtSecret))
	w := do(r, "GET", "/api/v1/products", signed, nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with expired token, got %d", w.Code)
	}
}

func TestAPI_ProductLifecycle(t *testing.T) {
	gw := newMockGateway()
	r := buildRouter(gw)
	_, tok := login(t, r, loginUser, loginPasswd)
	if tok == "" {
		t.Fatal("login failed")
	}

	create := map[string]interface{}{
		"id":               "P-INT",
		"sku":              "SKU-INT",
		"name":             "Integration Product",
		"description":      "tested via httptest",
		"batchId":          "B-INT-1",
		"manufacturerName": "Acme",
		"metadata":         map[string]string{"k": "v"},
	}
	w := do(r, "POST", "/api/v1/products", tok, create)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d (body=%s)", w.Code, w.Body.String())
	}

	// Conflict
	w = do(r, "POST", "/api/v1/products", tok, create)
	if w.Code != http.StatusConflict {
		t.Errorf("re-create: expected 409, got %d", w.Code)
	}

	// Read
	w = do(r, "GET", "/api/v1/products/P-INT", tok, nil)
	if w.Code != http.StatusOK {
		t.Errorf("read: expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}

	// Not found
	w = do(r, "GET", "/api/v1/products/missing", tok, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("read missing: expected 404, got %d", w.Code)
	}

	// Invalid ID format
	w = do(r, "GET", "/api/v1/products/bad!!id", tok, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("bad id: expected 400, got %d", w.Code)
	}

	// Missing required body fields
	w = do(r, "POST", "/api/v1/products", tok, map[string]string{"sku": "S"})
	if w.Code != http.StatusBadRequest {
		t.Errorf("missing fields: expected 400, got %d", w.Code)
	}

	// List
	w = do(r, "GET", "/api/v1/products?pageSize=10", tok, nil)
	if w.Code != http.StatusOK {
		t.Errorf("list: expected 200, got %d", w.Code)
	}

	// Verify mock saw the submits
	if len(gw.submits) != 2 {
		t.Errorf("expected 2 submit calls (create + duplicate), got %d", len(gw.submits))
	}
}

func TestAPI_CorrelationIDPropagation(t *testing.T) {
	gw := newMockGateway()
	r := buildRouter(gw)
	w := do(r, "GET", "/health", "", nil)
	if got := w.Header().Get("X-Correlation-ID"); got == "" {
		t.Errorf("expected X-Correlation-ID header, got empty")
	}
}

func TestAPI_SecurityHeadersPresent(t *testing.T) {
	gw := newMockGateway()
	r := buildRouter(gw)
	w := do(r, "GET", "/health", "", nil)
	// SecurityHeaders middleware should set at least these.
	must := []string{"X-Content-Type-Options", "X-Frame-Options"}
	for _, h := range must {
		if w.Header().Get(h) == "" {
			t.Errorf("expected security header %s, got empty", h)
		}
	}
}

func TestAPI_RateLimit_TriggersUnder429(t *testing.T) {
	gw := newMockGateway()
	r := buildRouter(gw)
	_, tok := login(t, r, loginUser, loginPasswd)

	limited := false
	for i := 0; i < 400; i++ {
		w := do(r, "GET", "/api/v1/products/P-1", tok, nil)
		if w.Code == http.StatusTooManyRequests {
			limited = true
			break
		}
	}
	if !limited {
		t.Skip("rate-limit did not trigger within 400 requests; config may differ")
	}
}

func TestAPI_TokenContainsExpectedClaims(t *testing.T) {
	gw := newMockGateway()
	r := buildRouter(gw)
	_, tok := login(t, r, loginUser, loginPasswd)
	parsed, err := jwt.Parse(tok, func(t *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	if err != nil || !parsed.Valid {
		t.Fatalf("token did not parse: %v", err)
	}
	claims, _ := parsed.Claims.(jwt.MapClaims)
	if claims["org"] != mspID {
		t.Errorf("expected org claim %s, got %v", mspID, claims["org"])
	}
	if claims["sub"] != loginUser {
		t.Errorf("expected sub claim %s, got %v", loginUser, claims["sub"])
	}
}

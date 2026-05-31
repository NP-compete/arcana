package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq"
)

const testSigningKey = "arcana-jwt-signing-key-change-in-production"

func stubDB() *sql.DB {
	db, _ := sql.Open("postgres", "host=localhost port=0 user=nobody dbname=nonexistent sslmode=disable connect_timeout=1")
	return db
}

func testDistributedRateLimiter() *DistributedRateLimiter {
	return &DistributedRateLimiter{local: NewRateLimiter()}
}

func buildTestJWT(t *testing.T, claims JWTClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(testSigningKey))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return tokenStr
}

func testClaims(subject, issuer, tenant string, roles []string, issuedAt, expiresAt time.Time) JWTClaims {
	return JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		Roles:  roles,
		Tenant: tenant,
	}
}

func TestAuthMiddleware_OpenMode(t *testing.T) {
	gw := &EnterpriseGateway{
		authMode:    AuthModeOpen,
		rateLimiter: testDistributedRateLimiter(),
	}
	gw.auditLog = &AuditLog{db: stubDB()}

	called := false
	checkIdentity := true
	inner := func(w http.ResponseWriter, r *http.Request) {
		called = true
		if checkIdentity {
			if r.Header.Get("X-Arcana-User") == "" {
				t.Error("expected X-Arcana-User header to be set")
			}
			if r.Header.Get("X-Arcana-Tenant") != "default" {
				t.Errorf("expected tenant default, got %q", r.Header.Get("X-Arcana-Tenant"))
			}
		}
		w.WriteHeader(http.StatusOK)
	}

	handler := gw.AuthMiddleware(inner)

	t.Run("no_auth_header_passes", func(t *testing.T) {
		called = false
		req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
		rec := httptest.NewRecorder()
		handler(rec, req)
		if !called {
			t.Error("inner handler was not called in open mode")
		}
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("health_endpoints_bypass", func(t *testing.T) {
		checkIdentity = false
		defer func() { checkIdentity = true }()
		for _, path := range []string{"/healthz", "/readyz", "/api/v1/version"} {
			called = false
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			handler(rec, req)
			if !called {
				t.Errorf("inner handler was not called for %s", path)
			}
		}
	})

	t.Run("options_request_passes", func(t *testing.T) {
		checkIdentity = false
		defer func() { checkIdentity = true }()
		called = false
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/agents", nil)
		rec := httptest.NewRecorder()
		handler(rec, req)
		if !called {
			t.Error("inner handler was not called for OPTIONS")
		}
	})

	t.Run("custom_role_header", func(t *testing.T) {
		called = false
		req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
		req.Header.Set("X-Arcana-Role", "developer")
		rec := httptest.NewRecorder()
		handler(rec, req)
		if !called {
			t.Error("inner handler was not called with custom role")
		}
		if roles := req.Header.Get("X-Arcana-Roles"); roles != "developer" {
			t.Errorf("expected roles=developer, got %q", roles)
		}
	})
}

func TestAuthMiddleware_APIKeyMode_MissingKey(t *testing.T) {
	gw := &EnterpriseGateway{
		authMode:    AuthModeAPIKey,
		rateLimiter: testDistributedRateLimiter(),
		auditLog:    &AuditLog{db: stubDB()},
	}
	inner := func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not be called without a key")
	}
	handler := gw.AuthMiddleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if !strings.Contains(body["error"], "API key required") {
		t.Errorf("unexpected error: %s", body["error"])
	}
}

func TestAuthMiddleware_APIKeyMode_InvalidKey(t *testing.T) {
	gw := &EnterpriseGateway{
		authMode:    AuthModeAPIKey,
		rateLimiter: testDistributedRateLimiter(),
		auditLog:    &AuditLog{db: stubDB()},
	}
	inner := func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not be called with bad key format")
	}
	handler := gw.AuthMiddleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	req.Header.Set("Authorization", "InvalidScheme")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestRBACCheck(t *testing.T) {
	tests := []struct {
		name     string
		roles    []string
		resource string
		verb     string
		want     bool
	}{
		{"admin_can_create_agents", []string{"admin"}, "agents", "create", true},
		{"admin_can_delete_tenants", []string{"admin"}, "tenants", "delete", true},
		{"admin_can_read_anything", []string{"admin"}, "whatever", "read", true},
		{"developer_can_create_agents", []string{"developer"}, "agents", "create", true},
		{"developer_can_read_models", []string{"developer"}, "models", "read", true},
		{"developer_cannot_delete_enterprise", []string{"developer"}, "enterprise", "delete", false},
		{"developer_cannot_read_enterprise", []string{"developer"}, "enterprise", "read", false},
		{"developer_cannot_read_audit", []string{"developer"}, "audit", "read", false},
		{"user_can_read_agents", []string{"user"}, "agents", "read", true},
		{"user_can_create_chat", []string{"user"}, "chat", "create", true},
		{"user_cannot_create_agents", []string{"user"}, "agents", "create", false},
		{"user_cannot_delete_anything", []string{"user"}, "agents", "delete", false},
		{"auditor_can_read_audit", []string{"auditor"}, "audit", "read", true},
		{"auditor_can_read_enterprise", []string{"auditor"}, "enterprise", "read", true},
		{"auditor_cannot_create_agents", []string{"auditor"}, "agents", "create", false},
		{"sre_can_read_agents", []string{"sre"}, "agents", "read", true},
		{"sre_can_update_agents", []string{"sre"}, "agents", "update", true},
		{"sre_cannot_create_agents", []string{"sre"}, "agents", "create", false},
		{"data_engineer_can_read_agents", []string{"data-engineer"}, "agents", "read", true},
		{"data_engineer_can_crud_connectors", []string{"data-engineer"}, "connectors", "create", true},
		{"data_engineer_cannot_create_agents", []string{"data-engineer"}, "agents", "create", false},
		{"unknown_role_denied", []string{"ghost"}, "agents", "read", false},
		{"multi_role_escalation", []string{"user", "admin"}, "tenants", "delete", true},
		{"no_roles_denied", []string{}, "agents", "read", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := hasPermission(tc.roles, tc.resource, tc.verb)
			if got != tc.want {
				t.Errorf("hasPermission(%v, %q, %q) = %v, want %v",
					tc.roles, tc.resource, tc.verb, got, tc.want)
			}
		})
	}
}

func TestRateLimiter(t *testing.T) {
	t.Run("allow_under_limit", func(t *testing.T) {
		rl := NewRateLimiter()
		for i := 0; i < 5; i++ {
			if !rl.Allow("user-1", 5) {
				t.Errorf("request %d should have been allowed", i)
			}
		}
	})
	t.Run("block_over_limit", func(t *testing.T) {
		rl := NewRateLimiter()
		for i := 0; i < 3; i++ {
			rl.Allow("user-2", 3)
		}
		if rl.Allow("user-2", 3) {
			t.Error("request should have been blocked (bucket empty)")
		}
	})
	t.Run("refills_over_time", func(t *testing.T) {
		rl := NewRateLimiter()
		for i := 0; i < 2; i++ {
			rl.Allow("user-3", 2)
		}
		if rl.Allow("user-3", 2) {
			t.Error("should be blocked immediately after draining")
		}
		time.Sleep(600 * time.Millisecond)
		if !rl.Allow("user-3", 2) {
			t.Error("request should be allowed after refill period")
		}
	})
	t.Run("independent_keys", func(t *testing.T) {
		rl := NewRateLimiter()
		for i := 0; i < 2; i++ {
			rl.Allow("key-a", 2)
		}
		if !rl.Allow("key-b", 2) {
			t.Error("key-b should not be affected by key-a")
		}
	})
}

func TestDistributedRateLimiter_Fallback(t *testing.T) {
	t.Run("fallback_allow_under_limit", func(t *testing.T) {
		rl := &DistributedRateLimiter{local: NewRateLimiter()}
		for i := 0; i < 5; i++ {
			if !rl.Allow("dist-user-1", 5) {
				t.Errorf("request %d should have been allowed via fallback", i)
			}
		}
	})
	t.Run("fallback_block_over_limit", func(t *testing.T) {
		rl := &DistributedRateLimiter{local: NewRateLimiter()}
		for i := 0; i < 3; i++ {
			rl.Allow("dist-user-2", 3)
		}
		if rl.Allow("dist-user-2", 3) {
			t.Error("request should have been blocked via fallback (bucket empty)")
		}
	})
	t.Run("fallback_independent_keys", func(t *testing.T) {
		rl := &DistributedRateLimiter{local: NewRateLimiter()}
		for i := 0; i < 2; i++ {
			rl.Allow("dist-key-a", 2)
		}
		if !rl.Allow("dist-key-b", 2) {
			t.Error("dist-key-b should not be affected by dist-key-a via fallback")
		}
	})
	t.Run("fallback_refills_over_time", func(t *testing.T) {
		rl := &DistributedRateLimiter{local: NewRateLimiter()}
		for i := 0; i < 2; i++ {
			rl.Allow("dist-user-3", 2)
		}
		if rl.Allow("dist-user-3", 2) {
			t.Error("should be blocked immediately after draining via fallback")
		}
		time.Sleep(600 * time.Millisecond)
		if !rl.Allow("dist-user-3", 2) {
			t.Error("request should be allowed after refill period via fallback")
		}
	})
}

func TestJWTValidation(t *testing.T) {
	gw := &EnterpriseGateway{}

	t.Run("valid_token", func(t *testing.T) {
		now := time.Now()
		token := buildTestJWT(t, testClaims(
			"test-user", "arcana", "acme",
			[]string{"developer"},
			now.Add(-10*time.Second), now.Add(3600*time.Second),
		))
		claims, err := gw.validateJWT(token)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if claims.Subject != "test-user" {
			t.Errorf("subject: got %q, want %q", claims.Subject, "test-user")
		}
		if claims.Tenant != "acme" {
			t.Errorf("tenant: got %q, want %q", claims.Tenant, "acme")
		}
		if len(claims.Roles) != 1 || claims.Roles[0] != "developer" {
			t.Errorf("roles: got %v, want [developer]", claims.Roles)
		}
	})

	t.Run("expired_token", func(t *testing.T) {
		now := time.Now()
		token := buildTestJWT(t, testClaims(
			"test-user", "arcana", "acme",
			[]string{"developer"},
			now.Add(-7200*time.Second), now.Add(-3600*time.Second),
		))
		_, err := gw.validateJWT(token)
		if err == nil {
			t.Fatal("expected error for expired token")
		}
		if !strings.Contains(err.Error(), "expired") {
			t.Errorf("error should mention 'expired', got: %v", err)
		}
	})

	t.Run("malformed_token_no_dots", func(t *testing.T) {
		_, err := gw.validateJWT("this-is-not-a-jwt")
		if err == nil {
			t.Fatal("expected error for malformed token")
		}
	})

	t.Run("malformed_token_two_parts", func(t *testing.T) {
		_, err := gw.validateJWT("header.payload")
		if err == nil {
			t.Fatal("expected error for two-part token")
		}
	})

	t.Run("bad_signature", func(t *testing.T) {
		now := time.Now()
		token := buildTestJWT(t, testClaims(
			"test-user", "arcana", "acme",
			[]string{"developer"},
			now.Add(-10*time.Second), now.Add(3600*time.Second),
		))
		parts := strings.Split(token, ".")
		parts[2] = "invalidsignature"
		tampered := strings.Join(parts, ".")
		_, err := gw.validateJWT(tampered)
		if err == nil {
			t.Fatal("expected error for bad signature")
		}
	})

	t.Run("missing_subject", func(t *testing.T) {
		now := time.Now()
		token := buildTestJWT(t, testClaims(
			"", "arcana", "acme",
			[]string{"developer"},
			now.Add(-10*time.Second), now.Add(3600*time.Second),
		))
		_, err := gw.validateJWT(token)
		if err == nil {
			t.Fatal("expected error for missing subject")
		}
		if !strings.Contains(err.Error(), "subject") {
			t.Errorf("error should mention 'subject', got: %v", err)
		}
	})

	t.Run("defaults_applied", func(t *testing.T) {
		now := time.Now()
		claims := JWTClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   "minimal-user",
				IssuedAt:  jwt.NewNumericDate(now.Add(-10 * time.Second)),
				ExpiresAt: jwt.NewNumericDate(now.Add(3600 * time.Second)),
			},
			Roles:  nil,
			Tenant: "",
		}
		token := buildTestJWT(t, claims)
		result, err := gw.validateJWT(token)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Tenant != "default" {
			t.Errorf("tenant: got %q, want %q", result.Tenant, "default")
		}
		if len(result.Roles) != 1 || result.Roles[0] != "user" {
			t.Errorf("roles: got %v, want [user]", result.Roles)
		}
	})
}

func TestRouteToResource(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/api/v1/agents", "agents"},
		{"/api/v1/agents/123", "agents"},
		{"/api/v1/tasks", "tasks"},
		{"/api/v1/enterprise/audit", "enterprise"},
		{"/api/v1/compliance", "compliance"},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			got := routeToResource(tc.path)
			if got != tc.want {
				t.Errorf("routeToResource(%q) = %q, want %q", tc.path, got, tc.want)
			}
		})
	}
}

func TestMethodToVerb(t *testing.T) {
	tests := []struct {
		method string
		want   string
	}{
		{http.MethodGet, "read"},
		{http.MethodPost, "create"},
		{http.MethodPut, "update"},
		{http.MethodPatch, "update"},
		{http.MethodDelete, "delete"},
		{"UNKNOWN", "read"},
	}
	for _, tc := range tests {
		t.Run(tc.method, func(t *testing.T) {
			got := methodToVerb(tc.method)
			if got != tc.want {
				t.Errorf("methodToVerb(%q) = %q, want %q", tc.method, got, tc.want)
			}
		})
	}
}

func TestHashKey(t *testing.T) {
	h1 := hashKey("my-secret-key")
	h2 := hashKey("my-secret-key")
	if h1 != h2 {
		t.Error("hashKey should be deterministic")
	}
	h3 := hashKey("different-key")
	if h1 == h3 {
		t.Error("different keys should produce different hashes")
	}
	if len(h1) != 64 {
		t.Errorf("expected 64-char hex string, got length %d", len(h1))
	}
}

func TestWriteJSONError(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSONError(rec, http.StatusForbidden, "access denied")
	if rec.Code != http.StatusForbidden {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusForbidden)
	}
	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("content-type: got %q, want %q", ct, "application/json")
	}
	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["error"] != "access denied" {
		t.Errorf("error: got %q, want %q", body["error"], "access denied")
	}
}

func TestAuthMiddleware_JWTMode(t *testing.T) {
	gw := &EnterpriseGateway{
		authMode:    AuthModeJWT,
		rateLimiter: testDistributedRateLimiter(),
		auditLog:    &AuditLog{db: stubDB()},
	}
	inner := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}
	handler := gw.AuthMiddleware(inner)

	t.Run("no_token_returns_401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
		rec := httptest.NewRecorder()
		handler(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("valid_jwt_passes", func(t *testing.T) {
		now := time.Now()
		token := buildTestJWT(t, testClaims(
			"jwt-user", "arcana", "default",
			[]string{"admin"},
			now.Add(-10*time.Second), now.Add(3600*time.Second),
		))
		req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		handler(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("expired_jwt_returns_401", func(t *testing.T) {
		now := time.Now()
		token := buildTestJWT(t, testClaims(
			"jwt-user", "arcana", "default",
			[]string{"admin"},
			now.Add(-7200*time.Second), now.Add(-3600*time.Second),
		))
		req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		handler(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("rbac_blocks_unauthorized_access", func(t *testing.T) {
		now := time.Now()
		token := buildTestJWT(t, testClaims(
			"limited-user", "arcana", "default",
			[]string{"user"},
			now.Add(-10*time.Second), now.Add(3600*time.Second),
		))
		req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		handler(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d; body: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestJWTRateLimiting(t *testing.T) {
	localRL := NewRateLimiter()
	gw := &EnterpriseGateway{
		authMode:    AuthModeJWT,
		rateLimiter: &DistributedRateLimiter{local: localRL},
		auditLog:    &AuditLog{db: stubDB()},
	}
	inner := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	handler := gw.AuthMiddleware(inner)

	now := time.Now()
	token := buildTestJWT(t, testClaims(
		"rate-test-user", "arcana", "default",
		[]string{"admin"},
		now.Add(-10*time.Second), now.Add(3600*time.Second),
	))

	// Pre-exhaust the rate limiter for this JWT subject (100 tokens at 100/s).
	for i := 0; i < 100; i++ {
		localRL.Allow("jwt:rate-test-user", 100)
	}

	// Next request should be rate-limited (429).
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 (rate limited), got %d; body: %s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if !strings.Contains(body["error"], "rate limit") {
		t.Errorf("expected rate limit error message, got: %s", body["error"])
	}
}

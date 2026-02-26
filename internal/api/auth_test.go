package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/config"
	"github.com/soochol/upal/internal/services"
	"github.com/soochol/upal/internal/upal"
)

const testJWTSecret = "test-jwt-secret-for-unit-tests"

func newTestAuthService() *services.AuthService {
	authCfg := config.AuthConfig{
		Google: config.OAuthProviderConfig{
			ClientID:     "test-google-client-id",
			ClientSecret: "test-google-client-secret",
		},
		GitHub: config.OAuthProviderConfig{
			ClientID:     "test-github-client-id",
			ClientSecret: "test-github-client-secret",
		},
		JWTSecret: testJWTSecret,
	}
	return services.NewAuthService(nil, authCfg, "http://localhost:8080")
}

func newTestAuthServer() *Server {
	srv := &Server{}
	srv.authSvc = newTestAuthService()
	return srv
}

// --- authLogin ---

func TestAuthLogin_RedirectsToGoogle(t *testing.T) {
	srv := newTestAuthServer()

	req := httptest.NewRequest("GET", "/api/auth/login/google", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("provider", "google")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	srv.authLogin(w, req)

	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusTemporaryRedirect)
	}

	location := w.Header().Get("Location")
	if !strings.Contains(location, "accounts.google.com") {
		t.Errorf("redirect should point to Google, got: %s", location)
	}

	// Verify state cookie was set
	var stateCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "oauth_state" {
			stateCookie = c
		}
	}
	if stateCookie == nil {
		t.Fatal("oauth_state cookie not set")
	}
	if !stateCookie.HttpOnly {
		t.Error("oauth_state cookie should be HttpOnly")
	}
	if stateCookie.Value == "" {
		t.Error("oauth_state cookie should have a value")
	}
}

func TestAuthLogin_RedirectsToGitHub(t *testing.T) {
	srv := newTestAuthServer()

	req := httptest.NewRequest("GET", "/api/auth/login/github", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("provider", "github")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	srv.authLogin(w, req)

	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusTemporaryRedirect)
	}

	location := w.Header().Get("Location")
	if !strings.Contains(location, "github.com") {
		t.Errorf("redirect should point to GitHub, got: %s", location)
	}
}

func TestAuthLogin_UnsupportedProvider(t *testing.T) {
	srv := newTestAuthServer()

	req := httptest.NewRequest("GET", "/api/auth/login/twitter", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("provider", "twitter")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	srv.authLogin(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAuthLogin_NilAuthService(t *testing.T) {
	srv := &Server{} // no authSvc

	req := httptest.NewRequest("GET", "/api/auth/login/google", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("provider", "google")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	srv.authLogin(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- authCallback ---

func TestAuthCallback_MissingStateCookie(t *testing.T) {
	srv := newTestAuthServer()

	req := httptest.NewRequest("GET", "/api/auth/callback/google?state=abc&code=xyz", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("provider", "google")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	srv.authCallback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "missing state cookie") {
		t.Errorf("body: got %q", w.Body.String())
	}
}

func TestAuthCallback_StateMismatch(t *testing.T) {
	srv := newTestAuthServer()

	req := httptest.NewRequest("GET", "/api/auth/callback/google?state=wrong&code=xyz", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "correct"})
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("provider", "google")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	srv.authCallback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "state mismatch") {
		t.Errorf("body: got %q", w.Body.String())
	}
}

func TestAuthCallback_MissingCode(t *testing.T) {
	srv := newTestAuthServer()

	req := httptest.NewRequest("GET", "/api/auth/callback/google?state=abc", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "abc"})
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("provider", "google")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	srv.authCallback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "missing code") {
		t.Errorf("body: got %q", w.Body.String())
	}
}

func TestAuthCallback_OAuthProviderError(t *testing.T) {
	srv := newTestAuthServer()

	req := httptest.NewRequest("GET", "/api/auth/callback/google?state=abc&error=access_denied", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "abc"})
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("provider", "google")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	srv.authCallback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "access_denied") {
		t.Errorf("body should contain error message, got %q", w.Body.String())
	}
}

func TestAuthCallback_NilAuthService(t *testing.T) {
	srv := &Server{}

	req := httptest.NewRequest("GET", "/api/auth/callback/google?state=abc&code=xyz", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("provider", "google")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	srv.authCallback(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- authRefresh ---

func TestAuthRefresh_MissingCookie(t *testing.T) {
	srv := newTestAuthServer()

	req := httptest.NewRequest("POST", "/api/auth/refresh", nil)
	w := httptest.NewRecorder()
	srv.authRefresh(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthRefresh_InvalidToken(t *testing.T) {
	srv := newTestAuthServer()

	req := httptest.NewRequest("POST", "/api/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "invalid-jwt-token"})

	w := httptest.NewRecorder()
	srv.authRefresh(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthRefresh_NilAuthService(t *testing.T) {
	srv := &Server{}

	req := httptest.NewRequest("POST", "/api/auth/refresh", nil)
	w := httptest.NewRecorder()
	srv.authRefresh(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- authLogout ---

func TestAuthLogout(t *testing.T) {
	srv := newTestAuthServer()

	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	w := httptest.NewRecorder()
	srv.authLogout(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusNoContent)
	}

	var refreshCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "refresh_token" {
			refreshCookie = c
		}
	}
	if refreshCookie == nil {
		t.Fatal("refresh_token cookie not set")
	}
	if refreshCookie.MaxAge != -1 {
		t.Errorf("refresh_token MaxAge: got %d, want -1", refreshCookie.MaxAge)
	}
	if refreshCookie.Value != "" {
		t.Errorf("refresh_token Value: got %q, want empty", refreshCookie.Value)
	}
}

// --- authMe ---

func TestAuthMe_Unauthenticated(t *testing.T) {
	srv := newTestAuthServer()

	// No user in context → defaults to "default"
	req := httptest.NewRequest("GET", "/api/auth/me", nil)
	w := httptest.NewRecorder()
	srv.authMe(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMe_DefaultUser(t *testing.T) {
	srv := newTestAuthServer()

	// Explicitly set "default" user in context
	ctx := upal.WithUserID(context.Background(), "default")
	req := httptest.NewRequest("GET", "/api/auth/me", nil).WithContext(ctx)

	w := httptest.NewRecorder()
	srv.authMe(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMe_NilAuthService(t *testing.T) {
	srv := &Server{}

	ctx := upal.WithUserID(context.Background(), "user-123")
	req := httptest.NewRequest("GET", "/api/auth/me", nil).WithContext(ctx)

	w := httptest.NewRecorder()
	srv.authMe(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- AuthMiddleware ---

func TestAuthMiddleware_NilAuthService_InjectsDefaultUser(t *testing.T) {
	var capturedUserID string
	handler := AuthMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = upal.UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/workflows", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}
	if capturedUserID != "default" {
		t.Errorf("userID: got %q, want %q", capturedUserID, "default")
	}
}

func TestAuthMiddleware_SkipsNonAPIPath(t *testing.T) {
	called := false
	handler := AuthMiddleware(newTestAuthService())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/static/app.js", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Fatal("handler should have been called for non-API path")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAuthMiddleware_SkipsAuthRoutes(t *testing.T) {
	called := false
	handler := AuthMiddleware(newTestAuthService())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/auth/login/google", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Fatal("handler should have been called for auth path")
	}
}

func TestAuthMiddleware_MissingAuthHeader(t *testing.T) {
	handler := AuthMiddleware(newTestAuthService())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called without auth header")
	}))

	req := httptest.NewRequest("GET", "/api/workflows", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	handler := AuthMiddleware(newTestAuthService())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called with invalid token")
	}))

	req := httptest.NewRequest("GET", "/api/workflows", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	authSvc := newTestAuthService()
	testUser := &upal.User{ID: "user-42", Email: "test@example.com", Name: "Test"}
	accessToken, _, err := authSvc.GenerateTokens(testUser)
	if err != nil {
		t.Fatalf("GenerateTokens: %v", err)
	}

	var capturedUserID string
	handler := AuthMiddleware(authSvc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = upal.UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/workflows", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}
	if capturedUserID != "user-42" {
		t.Errorf("userID: got %q, want %q", capturedUserID, "user-42")
	}
}

// --- Token generation/validation round-trip ---

func TestAuthService_TokenRoundTrip(t *testing.T) {
	authSvc := newTestAuthService()
	user := &upal.User{ID: "user-99", Email: "user@test.com"}

	accessToken, refreshToken, err := authSvc.GenerateTokens(user)
	if err != nil {
		t.Fatalf("GenerateTokens: %v", err)
	}

	// Validate access token
	userID, err := authSvc.ValidateAccessToken(accessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken: %v", err)
	}
	if userID != "user-99" {
		t.Errorf("access token userID: got %q, want %q", userID, "user-99")
	}

	// Validate refresh token
	userID, err = authSvc.ValidateRefreshToken(refreshToken)
	if err != nil {
		t.Fatalf("ValidateRefreshToken: %v", err)
	}
	if userID != "user-99" {
		t.Errorf("refresh token userID: got %q, want %q", userID, "user-99")
	}

	// Access token should NOT validate as refresh
	_, err = authSvc.ValidateRefreshToken(accessToken)
	if err == nil {
		t.Error("access token should not validate as refresh token")
	}

	// Refresh token should NOT validate as access
	_, err = authSvc.ValidateAccessToken(refreshToken)
	if err == nil {
		t.Error("refresh token should not validate as access token")
	}
}

// --- Integration: Full handler routing via Handler() ---

func TestAuth_RoutingViaHandler(t *testing.T) {
	srv := newTestServer()
	srv.authSvc = newTestAuthService()

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{"login google", "GET", "/api/auth/login/google", http.StatusTemporaryRedirect},
		{"login github", "GET", "/api/auth/login/github", http.StatusTemporaryRedirect},
		{"logout", "POST", "/api/auth/logout", http.StatusNoContent},
		{"refresh no cookie", "POST", "/api/auth/refresh", http.StatusUnauthorized},
		{"me no auth", "GET", "/api/auth/me", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			srv.Handler().ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status: got %d, want %d, body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestAuth_ProtectedRouteWithToken(t *testing.T) {
	srv := newTestServer()
	srv.authSvc = newTestAuthService()

	testUser := &upal.User{ID: "user-1", Email: "test@test.com"}
	accessToken, _, err := srv.authSvc.GenerateTokens(testUser)
	if err != nil {
		t.Fatalf("GenerateTokens: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/workflows", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var workflows []json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &workflows); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func TestAuth_ProtectedRouteWithoutToken(t *testing.T) {
	srv := newTestServer()
	srv.authSvc = newTestAuthService()

	req := httptest.NewRequest("GET", "/api/workflows", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

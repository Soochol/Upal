package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/upal"
)

const refreshTokenMaxAge = 7 * 24 * 60 * 60 // 7 days

func setRefreshTokenCookie(w http.ResponseWriter, value string) {
	maxAge := refreshTokenMaxAge
	var expires time.Time
	if value == "" {
		maxAge = -1
		expires = time.Unix(0, 0)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  expires,
	})
}

func (s *Server) requireAuth(w http.ResponseWriter) bool {
	if s.authSvc == nil {
		http.Error(w, "auth not configured", http.StatusNotFound)
		return false
	}
	return true
}

func (s *Server) authLogin(w http.ResponseWriter, r *http.Request) {
	if !s.requireAuth(w) {
		return
	}
	provider := chi.URLParam(r, "provider")

	oauthCfg, err := s.authSvc.OAuthConfig(provider)
	if err != nil {
		slog.Warn("oauth config error", "provider", provider, "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	state := generateState()
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, oauthCfg.AuthCodeURL(state), http.StatusTemporaryRedirect)
}

func (s *Server) authCallback(w http.ResponseWriter, r *http.Request) {
	if !s.requireAuth(w) {
		return
	}
	provider := chi.URLParam(r, "provider")

	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value == "" {
		http.Error(w, "missing state cookie", http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("state") != stateCookie.Value {
		http.Error(w, "state mismatch", http.StatusBadRequest)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		slog.Warn("oauth provider error", "provider", provider, "error", errMsg)
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	user, err := s.authSvc.ExchangeAndUpsertUser(r.Context(), provider, code)
	if err != nil {
		slog.Error("oauth exchange failed", "provider", provider, "err", err)
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}

	accessToken, refreshToken, err := s.authSvc.GenerateTokens(user)
	if err != nil {
		slog.Error("token generation failed", "user", user.ID, "err", err)
		http.Error(w, "token generation failed", http.StatusInternalServerError)
		return
	}

	setRefreshTokenCookie(w, refreshToken)
	http.Redirect(w, r, "/?token="+accessToken, http.StatusTemporaryRedirect)
}

func (s *Server) authRefresh(w http.ResponseWriter, r *http.Request) {
	if !s.requireAuth(w) {
		return
	}

	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie.Value == "" {
		http.Error(w, "missing refresh token", http.StatusUnauthorized)
		return
	}

	userID, err := s.authSvc.ValidateRefreshToken(cookie.Value)
	if err != nil {
		http.Error(w, "invalid refresh token", http.StatusUnauthorized)
		return
	}

	user, err := s.authSvc.GetUser(r.Context(), userID)
	if err != nil {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}

	accessToken, refreshToken, err := s.authSvc.GenerateTokens(user)
	if err != nil {
		http.Error(w, "token generation failed", http.StatusInternalServerError)
		return
	}

	setRefreshTokenCookie(w, refreshToken)
	writeJSON(w, map[string]string{"token": accessToken})
}

func (s *Server) authLogout(w http.ResponseWriter, _ *http.Request) {
	setRefreshTokenCookie(w, "")
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) authMe(w http.ResponseWriter, r *http.Request) {
	if !s.requireAuth(w) {
		return
	}

	userID := upal.UserIDFromContext(r.Context())
	if userID == "" || userID == "default" {
		http.Error(w, "not authenticated", http.StatusUnauthorized)
		return
	}

	user, err := s.authSvc.GetUser(r.Context(), userID)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	writeJSON(w, user)
}

func generateState() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	return hex.EncodeToString(b)
}

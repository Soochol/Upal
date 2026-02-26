package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/upal"
)

// authLogin redirects the user to the OAuth provider's authorization page.
func (s *Server) authLogin(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	if s.authSvc == nil {
		http.Error(w, `{"error":"auth not configured"}`, http.StatusNotFound)
		return
	}

	oauthCfg, err := s.authSvc.OAuthConfig(provider)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	// Generate and store state in a cookie for CSRF protection.
	state := generateState()
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   600, // 10 minutes
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	url := oauthCfg.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// authCallback handles the OAuth provider's callback after authorization.
func (s *Server) authCallback(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	if s.authSvc == nil {
		http.Error(w, `{"error":"auth not configured"}`, http.StatusNotFound)
		return
	}

	// Verify state cookie for CSRF protection.
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value == "" {
		http.Error(w, `{"error":"missing state cookie"}`, http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("state") != stateCookie.Value {
		http.Error(w, `{"error":"state mismatch"}`, http.StatusBadRequest)
		return
	}

	// Clear the state cookie.
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	// Check for error from OAuth provider.
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		http.Error(w, `{"error":"`+errMsg+`"}`, http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, `{"error":"missing code"}`, http.StatusBadRequest)
		return
	}

	user, err := s.authSvc.ExchangeAndUpsertUser(r.Context(), provider, code)
	if err != nil {
		http.Error(w, `{"error":"authentication failed"}`, http.StatusInternalServerError)
		return
	}

	accessToken, refreshToken, err := s.authSvc.GenerateTokens(user)
	if err != nil {
		http.Error(w, `{"error":"token generation failed"}`, http.StatusInternalServerError)
		return
	}

	// Set refresh token as httpOnly cookie.
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		MaxAge:   7 * 24 * 60 * 60, // 7 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to frontend with access token.
	http.Redirect(w, r, "/?token="+accessToken, http.StatusTemporaryRedirect)
}

// authRefresh generates new tokens from a valid refresh token cookie.
func (s *Server) authRefresh(w http.ResponseWriter, r *http.Request) {
	if s.authSvc == nil {
		http.Error(w, `{"error":"auth not configured"}`, http.StatusNotFound)
		return
	}

	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie.Value == "" {
		http.Error(w, `{"error":"missing refresh token"}`, http.StatusUnauthorized)
		return
	}

	userID, err := s.authSvc.ValidateRefreshToken(cookie.Value)
	if err != nil {
		http.Error(w, `{"error":"invalid refresh token"}`, http.StatusUnauthorized)
		return
	}

	user, err := s.authSvc.GetUser(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusUnauthorized)
		return
	}

	accessToken, refreshToken, err := s.authSvc.GenerateTokens(user)
	if err != nil {
		http.Error(w, `{"error":"token generation failed"}`, http.StatusInternalServerError)
		return
	}

	// Set new refresh token cookie.
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		MaxAge:   7 * 24 * 60 * 60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	writeJSON(w, map[string]string{
		"access_token": accessToken,
	})
}

// authLogout clears the refresh token cookie.
func (s *Server) authLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
	})
	w.WriteHeader(http.StatusNoContent)
}

// authMe returns the current authenticated user's info.
func (s *Server) authMe(w http.ResponseWriter, r *http.Request) {
	if s.authSvc == nil {
		http.Error(w, `{"error":"auth not configured"}`, http.StatusNotFound)
		return
	}

	userID := upal.UserIDFromContext(r.Context())
	if userID == "" || userID == "default" {
		http.Error(w, `{"error":"not authenticated"}`, http.StatusUnauthorized)
		return
	}

	user, err := s.authSvc.GetUser(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}

	writeJSON(w, user)
}

// generateState creates a random hex string for CSRF protection.
func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

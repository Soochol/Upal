package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/soochol/upal/internal/config"
	"github.com/soochol/upal/internal/db"
	"github.com/soochol/upal/internal/upal"
	"golang.org/x/oauth2"
	oauthgithub "golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

const (
	accessTokenTTL  = 1 * time.Hour
	refreshTokenTTL = 30 * 24 * time.Hour
)

type tokenClaims struct {
	UserID string
	JTI    string
}

type AuthService struct {
	database  *db.DB
	authCfg   config.AuthConfig
	baseURL   string
	jwtSecret []byte
}

func NewAuthService(database *db.DB, authCfg config.AuthConfig, baseURL string) *AuthService {
	secret := authCfg.JWTSecret
	if secret == "" {
		secret = loadOrCreateJWTSecret("data/jwt_secret")
	}
	return &AuthService{
		database:  database,
		authCfg:   authCfg,
		baseURL:   baseURL,
		jwtSecret: []byte(secret),
	}
}

func loadOrCreateJWTSecret(path string) string {
	data, err := os.ReadFile(path)
	if err == nil {
		secret := strings.TrimSpace(string(data))
		if len(secret) >= 32 {
			slog.Info("loaded JWT secret from file", "path", path)
			return secret
		}
		if len(secret) > 0 {
			slog.Warn("JWT secret too short, generating new one", "path", path, "length", len(secret))
		}
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	secret := hex.EncodeToString(b)

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		slog.Warn("cannot create directory for JWT secret", "path", path, "err", err)
		return secret
	}
	if err := os.WriteFile(path, []byte(secret+"\n"), 0o600); err != nil {
		slog.Warn("cannot persist JWT secret", "path", path, "err", err)
	} else {
		slog.Info("generated and persisted new JWT secret", "path", path)
	}
	return secret
}

func (s *AuthService) Enabled() bool {
	return s.authCfg.Google.IsConfigured() || s.authCfg.GitHub.IsConfigured()
}

func (s *AuthService) OAuthConfig(provider string) (*oauth2.Config, error) {
	switch provider {
	case "google":
		if !s.authCfg.Google.IsConfigured() {
			return nil, fmt.Errorf("google oauth not configured")
		}
		return &oauth2.Config{
			ClientID:     s.authCfg.Google.ClientID,
			ClientSecret: s.authCfg.Google.ClientSecret,
			Endpoint:     google.Endpoint,
			RedirectURL:  s.baseURL + "/api/auth/callback/google",
			Scopes:       []string{"openid", "email", "profile"},
		}, nil
	case "github":
		if !s.authCfg.GitHub.IsConfigured() {
			return nil, fmt.Errorf("github oauth not configured")
		}
		return &oauth2.Config{
			ClientID:     s.authCfg.GitHub.ClientID,
			ClientSecret: s.authCfg.GitHub.ClientSecret,
			Endpoint:     oauthgithub.Endpoint,
			RedirectURL:  s.baseURL + "/api/auth/callback/github",
			Scopes:       []string{"user:email"},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported oauth provider: %s", provider)
	}
}

func (s *AuthService) ExchangeAndUpsertUser(ctx context.Context, provider, code string) (*upal.User, error) {
	oauthCfg, err := s.OAuthConfig(provider)
	if err != nil {
		return nil, err
	}

	token, err := oauthCfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}

	var user *upal.User
	switch provider {
	case "google":
		user, err = s.fetchGoogleUser(ctx, token.AccessToken)
	case "github":
		user, err = s.fetchGitHubUser(ctx, token.AccessToken)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
	if err != nil {
		return nil, fmt.Errorf("fetch user info: %w", err)
	}

	user.OAuthProvider = provider
	return s.database.UpsertUser(ctx, user)
}

type googleUserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func (s *AuthService) fetchGoogleUser(ctx context.Context, accessToken string) (*upal.User, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google userinfo returned status %d", resp.StatusCode)
	}

	var info googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	return &upal.User{
		Email:     info.Email,
		Name:      info.Name,
		AvatarURL: info.Picture,
		OAuthID:   info.ID,
	}, nil
}

type githubUserInfo struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type githubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func (s *AuthService) fetchGitHubUser(ctx context.Context, accessToken string) (*upal.User, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github user returned status %d", resp.StatusCode)
	}

	var info githubUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	name := info.Name
	if name == "" {
		name = info.Login
	}

	email := info.Email
	if email == "" {
		var emailErr error
		email, emailErr = s.fetchGitHubPrimaryEmail(ctx, accessToken)
		if emailErr != nil || email == "" {
			return nil, fmt.Errorf("github email unavailable: %w", emailErr)
		}
	}

	return &upal.User{
		Email:     email,
		Name:      name,
		AvatarURL: info.AvatarURL,
		OAuthID:   fmt.Sprintf("%d", info.ID),
	}, nil
}

func (s *AuthService) fetchGitHubPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github emails returned status %d", resp.StatusCode)
	}

	var emails []githubEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}
	for _, e := range emails {
		if e.Primary {
			return e.Email, nil
		}
	}
	if len(emails) > 0 {
		return emails[0].Email, nil
	}
	return "", fmt.Errorf("no email found")
}

func (s *AuthService) GenerateTokens(ctx context.Context, user *upal.User, deviceInfo string) (access, refresh string, err error) {
	access, err = s.signToken(user.ID, "access", "", accessTokenTTL)
	if err != nil {
		return "", "", err
	}

	jti := upal.GenerateID("rt")
	refresh, err = s.signToken(user.ID, "refresh", jti, refreshTokenTTL)
	if err != nil {
		return "", "", err
	}

	if s.database != nil {
		now := time.Now()
		if dbErr := s.database.CreateRefreshToken(ctx, &db.RefreshToken{
			ID:         jti,
			UserID:     user.ID,
			TokenHash:  db.HashToken(refresh),
			DeviceInfo: deviceInfo,
			CreatedAt:  now,
			ExpiresAt:  now.Add(refreshTokenTTL),
		}); dbErr != nil {
			return "", "", fmt.Errorf("store refresh token: %w", dbErr)
		}
	}

	return access, refresh, nil
}

func (s *AuthService) signToken(userID, tokenType, jti string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":  userID,
		"type": tokenType,
		"iat":  now.Unix(),
		"exp":  now.Add(ttl).Unix(),
	}
	if jti != "" {
		claims["jti"] = jti
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sign %s token: %w", tokenType, err)
	}
	return signed, nil
}

func (s *AuthService) ValidateAccessToken(tokenStr string) (string, error) {
	claims, err := s.validateToken(tokenStr, "access")
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

func (s *AuthService) ValidateRefreshToken(tokenStr string) (*tokenClaims, error) {
	return s.validateToken(tokenStr, "refresh")
}

func (s *AuthService) validateToken(tokenStr, expectedType string) (*tokenClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	tokenType, _ := claims["type"].(string)
	if tokenType != expectedType {
		return nil, fmt.Errorf("invalid token type: expected %s, got %s", expectedType, tokenType)
	}

	userID, _ := claims["sub"].(string)
	if userID == "" {
		return nil, fmt.Errorf("missing user ID in token")
	}

	jti, _ := claims["jti"].(string)

	return &tokenClaims{UserID: userID, JTI: jti}, nil
}

func (s *AuthService) GetUser(ctx context.Context, id string) (*upal.User, error) {
	user, err := s.database.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user not found: %s", id)
	}
	return user, nil
}

// RotateRefreshToken validates the old refresh token, revokes it, and issues a new token pair.
// If a revoked token is reused, all tokens for the user are revoked (family revocation).
func (s *AuthService) RotateRefreshToken(ctx context.Context, refreshTokenStr, deviceInfo string) (string, string, error) {
	claims, err := s.ValidateRefreshToken(refreshTokenStr)
	if err != nil {
		return "", "", fmt.Errorf("invalid refresh token: %w", err)
	}

	// No DB — fall back to simple JWT-only validation (backward compat for tests)
	if s.database == nil {
		user := &upal.User{ID: claims.UserID}
		return s.GenerateTokens(ctx, user, deviceInfo)
	}

	if claims.JTI == "" {
		return "", "", fmt.Errorf("missing jti in refresh token")
	}

	stored, err := s.database.GetRefreshToken(ctx, claims.JTI)
	if err != nil {
		return "", "", fmt.Errorf("lookup refresh token: %w", err)
	}
	if stored == nil {
		return "", "", fmt.Errorf("refresh token not found")
	}

	// Token reuse detection: if already revoked, revoke entire family
	if stored.RevokedAt != nil {
		if err := s.database.RevokeAllRefreshTokens(ctx, stored.UserID); err != nil {
			slog.Error("family revocation failed", "user", stored.UserID, "err", err)
		}
		return "", "", fmt.Errorf("refresh token reuse detected")
	}

	// Verify hash matches
	if db.HashToken(refreshTokenStr) != stored.TokenHash {
		return "", "", fmt.Errorf("refresh token hash mismatch")
	}

	// Get user and generate new token pair
	user, err := s.GetUser(ctx, stored.UserID)
	if err != nil {
		return "", "", fmt.Errorf("get user for rotation: %w", err)
	}

	newAccess, newRefresh, err := s.GenerateTokens(ctx, user, deviceInfo)
	if err != nil {
		return "", "", fmt.Errorf("generate rotated tokens: %w", err)
	}

	// Extract new JTI from the new refresh token to record as replacement
	newClaims, err := s.ValidateRefreshToken(newRefresh)
	if err != nil {
		return "", "", fmt.Errorf("validate new refresh token: %w", err)
	}

	// Revoke old token, pointing to its replacement
	if err := s.database.RevokeRefreshToken(ctx, claims.JTI, newClaims.JTI); err != nil {
		return "", "", fmt.Errorf("revoke old refresh token: %w", err)
	}

	return newAccess, newRefresh, nil
}

// RevokeUserRefreshToken revokes a single refresh token by its JWT string.
func (s *AuthService) RevokeUserRefreshToken(refreshTokenStr string) error {
	claims, err := s.ValidateRefreshToken(refreshTokenStr)
	if err != nil {
		return nil
	}
	if s.database == nil || claims.JTI == "" {
		return nil
	}
	if err := s.database.RevokeRefreshToken(context.Background(), claims.JTI, ""); err != nil {
		slog.Warn("failed to revoke refresh token on logout", "jti", claims.JTI, "err", err)
	}
	return nil
}

// StartCleanup starts a background goroutine that periodically removes expired refresh tokens.
func (s *AuthService) StartCleanup(ctx context.Context) {
	if s.database == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				deleted, err := s.database.CleanupExpiredRefreshTokens(ctx)
				if err != nil {
					slog.Error("refresh token cleanup failed", "err", err)
				} else if deleted > 0 {
					slog.Info("cleaned up expired refresh tokens", "count", deleted)
				}
			}
		}
	}()
}

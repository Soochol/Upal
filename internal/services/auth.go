package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/soochol/upal/internal/config"
	"github.com/soochol/upal/internal/db"
	"github.com/soochol/upal/internal/upal"
	"golang.org/x/oauth2"
	oauthgithub "golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

// AuthService handles OAuth2 authentication and JWT token management.
type AuthService struct {
	database  *db.DB
	authCfg   config.AuthConfig
	baseURL   string
	jwtSecret []byte
}

// NewAuthService creates a new AuthService.
func NewAuthService(database *db.DB, authCfg config.AuthConfig, baseURL string) *AuthService {
	secret := authCfg.JWTSecret
	if secret == "" {
		b := make([]byte, 32)
		rand.Read(b)
		secret = hex.EncodeToString(b)
	}
	return &AuthService{
		database:  database,
		authCfg:   authCfg,
		baseURL:   baseURL,
		jwtSecret: []byte(secret),
	}
}

// Enabled returns true if at least one OAuth provider is configured.
func (s *AuthService) Enabled() bool {
	if s.authCfg.Google.ClientID != "" && s.authCfg.Google.ClientSecret != "" {
		return true
	}
	if s.authCfg.GitHub.ClientID != "" && s.authCfg.GitHub.ClientSecret != "" {
		return true
	}
	return false
}

// OAuthConfig returns the oauth2.Config for the given provider.
func (s *AuthService) OAuthConfig(provider string) (*oauth2.Config, error) {
	switch provider {
	case "google":
		if s.authCfg.Google.ClientID == "" {
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
		if s.authCfg.GitHub.ClientID == "" {
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

// ExchangeAndUpsertUser exchanges the OAuth code for a token, fetches user info,
// and upserts the user in the database.
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

// googleUserInfo represents the response from Google's userinfo endpoint.
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

// githubUserInfo represents the response from GitHub's user endpoint.
type githubUserInfo struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// githubEmail represents an email from GitHub's /user/emails endpoint.
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
		email, _ = s.fetchGitHubPrimaryEmail(ctx, accessToken)
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

// GenerateTokens creates an access token (30min) and refresh token (7d) for the given user.
func (s *AuthService) GenerateTokens(user *upal.User) (access, refresh string, err error) {
	now := time.Now()

	accessClaims := jwt.MapClaims{
		"sub":  user.ID,
		"type": "access",
		"iat":  now.Unix(),
		"exp":  now.Add(30 * time.Minute).Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	access, err = accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return "", "", fmt.Errorf("sign access token: %w", err)
	}

	refreshClaims := jwt.MapClaims{
		"sub":  user.ID,
		"type": "refresh",
		"iat":  now.Unix(),
		"exp":  now.Add(7 * 24 * time.Hour).Unix(),
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refresh, err = refreshToken.SignedString(s.jwtSecret)
	if err != nil {
		return "", "", fmt.Errorf("sign refresh token: %w", err)
	}

	return access, refresh, nil
}

// ValidateAccessToken validates a JWT access token and returns the user ID.
// Refresh tokens are rejected.
func (s *AuthService) ValidateAccessToken(tokenStr string) (string, error) {
	return s.validateToken(tokenStr, "access")
}

// ValidateRefreshToken validates a JWT refresh token and returns the user ID.
// Only refresh tokens are accepted.
func (s *AuthService) ValidateRefreshToken(tokenStr string) (string, error) {
	return s.validateToken(tokenStr, "refresh")
}

func (s *AuthService) validateToken(tokenStr, expectedType string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return "", fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	tokenType, _ := claims["type"].(string)
	if tokenType != expectedType {
		return "", fmt.Errorf("invalid token type: expected %s, got %s", expectedType, tokenType)
	}

	userID, _ := claims["sub"].(string)
	if userID == "" {
		return "", fmt.Errorf("missing user ID in token")
	}

	return userID, nil
}

// GetUser retrieves a user by ID from the database.
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

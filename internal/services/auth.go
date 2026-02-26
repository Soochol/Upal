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

const (
	accessTokenTTL  = 30 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
)

type AuthService struct {
	database  *db.DB
	authCfg   config.AuthConfig
	baseURL   string
	jwtSecret []byte
}

func NewAuthService(database *db.DB, authCfg config.AuthConfig, baseURL string) *AuthService {
	secret := authCfg.JWTSecret
	if secret == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			panic(fmt.Sprintf("crypto/rand failed: %v", err))
		}
		secret = hex.EncodeToString(b)
	}
	return &AuthService{
		database:  database,
		authCfg:   authCfg,
		baseURL:   baseURL,
		jwtSecret: []byte(secret),
	}
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

func (s *AuthService) GenerateTokens(user *upal.User) (access, refresh string, err error) {
	access, err = s.signToken(user.ID, "access", accessTokenTTL)
	if err != nil {
		return "", "", err
	}
	refresh, err = s.signToken(user.ID, "refresh", refreshTokenTTL)
	if err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

func (s *AuthService) signToken(userID, tokenType string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":  userID,
		"type": tokenType,
		"iat":  now.Unix(),
		"exp":  now.Add(ttl).Unix(),
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sign %s token: %w", tokenType, err)
	}
	return signed, nil
}

func (s *AuthService) ValidateAccessToken(tokenStr string) (string, error) {
	return s.validateToken(tokenStr, "access")
}

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

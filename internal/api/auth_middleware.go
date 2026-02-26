package api

import (
	"net/http"
	"strings"

	"github.com/soochol/upal/internal/services"
	"github.com/soochol/upal/internal/upal"
)

// AuthMiddleware returns a Chi middleware that validates Bearer tokens on API requests.
// Auth routes (/api/auth/*) and non-API paths (static files) are skipped.
// If authSvc is nil or auth is not enabled, a default user ID is injected.
func AuthMiddleware(authSvc *services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			// Skip non-API paths (static files).
			if !strings.HasPrefix(path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}

			// Skip auth routes.
			if strings.HasPrefix(path, "/api/auth/") {
				next.ServeHTTP(w, r)
				return
			}

			// If authSvc is nil or auth not enabled, use default user.
			if authSvc == nil || !authSvc.Enabled() {
				ctx := upal.WithUserID(r.Context(), "default")
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Extract Bearer token from Authorization header.
			auth := r.Header.Get("Authorization")
			if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
				http.Error(w, `{"error":"missing or invalid authorization header"}`, http.StatusUnauthorized)
				return
			}
			tokenStr := strings.TrimPrefix(auth, "Bearer ")

			userID, err := authSvc.ValidateAccessToken(tokenStr)
			if err != nil {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			ctx := upal.WithUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

package api

import (
	"net/http"
	"strings"

	"github.com/soochol/upal/internal/services"
	"github.com/soochol/upal/internal/upal"
)

// AuthMiddleware validates Bearer tokens on API requests.
// Non-API paths and /api/auth/* are skipped. When auth is disabled, a default user ID is injected.
func AuthMiddleware(authSvc *services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if !strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/api/auth/") {
				next.ServeHTTP(w, r)
				return
			}

			if authSvc == nil || !authSvc.Enabled() {
				ctx := upal.WithUserID(r.Context(), "default")
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				http.Error(w, `{"error":"missing or invalid authorization header"}`, http.StatusUnauthorized)
				return
			}

			userID, err := authSvc.ValidateAccessToken(strings.TrimPrefix(auth, "Bearer "))
			if err != nil {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			ctx := upal.WithUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

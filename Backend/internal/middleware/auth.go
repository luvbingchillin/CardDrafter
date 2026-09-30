package middleware

import (
	"backend/internal/utils"
	"context"
	"net/http"
)

// Define a custom type for context keys to avoid collisions
type contextKey string

const UserIDKey contextKey = "userID"

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Read the cookie or header
		cookie, err := r.Cookie("token")
		if err != nil {
			http.Error(w, "Unauthorized: missing token", http.StatusUnauthorized)
			return
		}

		// 2. Validate token
		claims, err := utils.ValidateToken(cookie.Value)
		if err != nil {
			http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
			return
		}

		// 3. Inject user info into the request context
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)

		// 4. Pass control to the next handler with the new context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

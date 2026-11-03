package middleware

import (
	"backend/internal/utils"
	"context"
	"crypto/rand"
	"encoding/hex"
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


// OptionalAuthMiddleware inspects requests for an authenticated JWT token cookie.
// If valid credentials exist, it injects the registered UserID into context.
// If credentials are absent or invalid, it falls back to a query parameter or generates
// an ephemeral guest identifier without rejecting the request with 401 Unauthorized.
func OptionalAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var userID string

		// 1. Attempt to extract and validate JWT cookie
		cookie, err := r.Cookie("token")
		if err == nil {
			claims, err := utils.ValidateToken(cookie.Value)
			if err == nil && claims.UserID != "" {
				userID = claims.UserID
			}
		}

		// 2. Fallback: inspect query parameter (e.g. ?userId=Guest_XXXX)
		if userID == "" {
			userID = r.URL.Query().Get("userId")
		}

		// 3. Fallback: generate a random ephemeral guest identifier
		if userID == "" {
			bytes := make([]byte, 4)
			if _, err := rand.Read(bytes); err == nil {
				userID = "guest_" + hex.EncodeToString(bytes)
			} else {
				userID = "guest_anon"
			}
		}

		// 4. Inject resolved user identity into context and continue
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

package middleware

import (
	"backend/internal/models"
	"backend/internal/utils"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
)

// Define a custom type for context keys to avoid collisions.
type contextKey string

const UserIDKey contextKey = "userID"

func writeAuthError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(models.ErrorResponse{Error: message})
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("token")
		if err != nil {
			writeAuthError(w, "unauthorized: missing token", http.StatusUnauthorized)
			return
		}

		claims, err := utils.ValidateToken(cookie.Value)
		if err != nil || claims.UserID == "" {
			writeAuthError(w, "unauthorized: invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// isValidGuestID ensures unauthenticated query parameter identities cannot spoof
// registered user IDs (e.g. MongoDB ObjectIDs) and must carry an explicit Guest_ prefix.
func isValidGuestID(id string) bool {
	if len(id) < 7 || len(id) > 32 {
		return false
	}
	return strings.HasPrefix(id, "Guest_") || strings.HasPrefix(id, "guest_")
}

// OptionalAuthMiddleware inspects requests for an authenticated JWT token cookie.
// If valid credentials exist, it injects the registered UserID into context.
// If credentials are absent or invalid, it accepts only guest-prefixed query IDs or
// generates a fresh ephemeral guest identifier.
func OptionalAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var userID string

		// 1. Attempt to extract and validate JWT cookie
		if cookie, err := r.Cookie("token"); err == nil {
			if claims, err := utils.ValidateToken(cookie.Value); err == nil && claims.UserID != "" {
				userID = claims.UserID
			}
		}

		// 2. Fallback: inspect query parameter, strictly restricted to Guest_ prefix to prevent seat hijacking
		if userID == "" {
			candidate := strings.TrimSpace(r.URL.Query().Get("userId"))
			if isValidGuestID(candidate) {
				userID = candidate
			}
		}

		// 3. Fallback: generate a random ephemeral guest identifier
		if userID == "" {
			bytes := make([]byte, 4)
			if _, err := rand.Read(bytes); err == nil {
				userID = "Guest_" + strings.ToUpper(hex.EncodeToString(bytes))
			} else {
				userID = "Guest_ANON"
			}
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

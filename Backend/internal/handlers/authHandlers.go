package handlers

import (
	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/utils"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// UserAuthenticator defines the authentication service contract required by AuthHandler.
type UserAuthenticator interface {
	Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error)
	Register(ctx context.Context, req models.RegisterRequest) (*models.AuthResponse, error)
	ProcessGoogleUser(ctx context.Context, googleID, email, name string) (*models.User, error)
	GetUserByID(ctx context.Context, id string) (*models.UserSummary, error)
}

type AuthHandler struct {
	authService       UserAuthenticator
	googleOAuthConfig *oauth2.Config
}

func NewAuthHandler(authService UserAuthenticator) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		googleOAuthConfig: &oauth2.Config{
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		},
	}
}

func isProductionEnv() bool {
	return os.Getenv("ENV") == "production"
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "invalid request payload", http.StatusBadRequest)
		return
	}
	response, err := h.authService.Login(r.Context(), req)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusUnauthorized)
		return
	}

	token, err := utils.GenerateToken(response.User.ID)
	if err != nil {
		slog.Error("Failed to generate JWT on login", "user_id", response.User.ID, "err", err)
		writeJSONError(w, "token generation failed", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,
		Secure:   isProductionEnv(),
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "invalid request payload", http.StatusBadRequest)
		return
	}
	response, err := h.authService.Register(r.Context(), req)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	token, err := utils.GenerateToken(response.User.ID)
	if err != nil {
		slog.Error("Failed to generate JWT on register", "user_id", response.User.ID, "err", err)
		writeJSONError(w, "token generation failed", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,
		Secure:   isProductionEnv(),
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		HttpOnly: true,
		Secure:   isProductionEnv(),
		Path:     "/",
		MaxAge:   -1,
	})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Logged out successfully"})
}

func (h *AuthHandler) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		slog.Error("Failed to generate OAuth state entropy", "err", err)
		writeJSONError(w, "failed to initialize OAuth flow", http.StatusInternalServerError)
		return
	}
	state := hex.EncodeToString(b)

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		HttpOnly: true,
		Secure:   isProductionEnv(),
		Path:     "/",
		MaxAge:   300,
		SameSite: http.SameSiteLaxMode,
	})

	url := h.googleOAuthConfig.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	oauthState, err := r.Cookie("oauth_state")
	if err != nil || r.FormValue("state") != oauthState.Value {
		writeJSONError(w, "invalid oauth state", http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")
	token, err := h.googleOAuthConfig.Exchange(r.Context(), code)
	if err != nil {
		slog.Error("Google OAuth code exchange failed", "err", err)
		writeJSONError(w, "oauth code exchange failed", http.StatusInternalServerError)
		return
	}

	// Use context-bound OAuth2 HTTP client (sends Authorization: Bearer header & honors context deadlines)
	oauthClient := h.googleOAuthConfig.Client(r.Context(), token)
	resp, err := oauthClient.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		slog.Error("Failed fetching Google userinfo", "err", err)
		writeJSONError(w, "failed retrieving Google user profile", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var googleUser struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		slog.Error("Failed decoding Google userinfo payload", "err", err)
		writeJSONError(w, "failed decoding Google user profile", http.StatusInternalServerError)
		return
	}

	user, err := h.authService.ProcessGoogleUser(r.Context(), googleUser.ID, googleUser.Email, googleUser.Name)
	if err != nil {
		slog.Error("Failed processing Google user", "email", googleUser.Email, "err", err)
		writeJSONError(w, "google authentication failed", http.StatusInternalServerError)
		return
	}

	jwtToken, err := utils.GenerateToken(user.ID.Hex())
	if err != nil {
		slog.Error("Failed generating JWT for Google user", "user_id", user.ID.Hex(), "err", err)
		writeJSONError(w, "failed generating token", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    jwtToken,
		HttpOnly: true,
		Secure:   isProductionEnv(),
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost"
	}
	http.Redirect(w, r, frontendURL, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		writeJSONError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.authService.GetUserByID(r.Context(), userID)
	if err != nil || user == nil {
		writeJSONError(w, "user not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(user)
}

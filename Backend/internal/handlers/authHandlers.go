package handlers

import (
	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/utils"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
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

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "invalid request payload"})
		return
	}
	response, err := h.authService.Login(r.Context(), req)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: err.Error()})
		return
	}

	token, err := utils.GenerateToken(response.User.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "token generation failed"})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,
		Secure:   false, // Set to true in production over HTTPS
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "invalid request payload"})
		return
	}
	response, err := h.authService.Register(r.Context(), req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: err.Error()})
		return
	}

	token, err := utils.GenerateToken(response.User.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "token generation failed"})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,
		Secure:   false,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		HttpOnly: true,
		Secure:   false,
		Path:     "/",
		MaxAge:   -1,
	})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Logged out successfully"})
}

func (h *AuthHandler) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	b := make([]byte, 16)
	rand.Read(b)
	state := hex.EncodeToString(b)

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		HttpOnly: true,
		Secure:   false,
		Path:     "/",
		MaxAge:   300,
	})

	url := h.googleOAuthConfig.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	oauthState, err := r.Cookie("oauth_state")
	if err != nil || r.FormValue("state") != oauthState.Value {
		http.Error(w, "invalid oauth state", http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")
	token, err := h.googleOAuthConfig.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "code exchange failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		http.Error(w, "failed getting user info: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var googleUser struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		http.Error(w, "failed reading user info: "+err.Error(), http.StatusInternalServerError)
		return
	}

	user, err := h.authService.ProcessGoogleUser(r.Context(), googleUser.ID, googleUser.Email, googleUser.Name)
	if err != nil {
		http.Error(w, "login failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	jwtToken, err := utils.GenerateToken(user.ID.Hex())
	if err != nil {
		http.Error(w, "failed generating token", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    jwtToken,
		HttpOnly: true,
		Secure:   false,
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
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "unauthorized"})
		return
	}

	user, err := h.authService.GetUserByID(r.Context(), userID)
	if err != nil || user == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "user not found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid" // New dependency for generating UUIDs
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// --- Configuration ---
const (
	// Replace with your actual Meta App ID and Secret obtained from Meta Developers.
	META_CLIENT_ID     = "YOUR_META_APP_ID"
	META_CLIENT_SECRET = "YOUR_META_APP_SECRET"
	META_REDIRECT_URI  = "http://localhost:8081/oauth/meta/callback"
	
	// Replace with your actual TikTok App Key and Secret.
	TIKTOK_CLIENT_KEY     = "YOUR_TIKTOK_CLIENT_KEY"
	TIKTOK_CLIENT_SECRET  = "YOUR_TIKTOK_CLIENT_SECRET"
	TIKTOK_REDIRECT_URI   = "http://localhost:8081/oauth/tiktok/callback"

	// Replace with your actual Snapchat Client ID and Secret.
	SNAPCHAT_CLIENT_ID     = "YOUR_SNAPCHAT_CLIENT_ID"
	SNAPCHAT_CLIENT_SECRET = "YOUR_SNAPCHAT_CLIENT_SECRET"
	SNAPCHAT_REDIRECT_URI  = "http://localhost:8081/oauth/snapchat/callback"
	
	// JWT Secret (generate a strong, random key in production)
	JWT_SECRET = "supersecretjwtkeythatshouldbeverylongandrandom"
	
	// URL for the Account Service
	ACCOUNT_SERVICE_URL = "http://localhost:8082"
)

// --- Database Connection ---
var db *sql.DB

// initDB connects to the PostgreSQL database and creates tables.
func initDB() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Auth Service successfully connected to PostgreSQL!")
	createTables()
}

// createTables runs SQL to set up the necessary database schema for the Auth service.
func createTables() {
	userTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		name TEXT,
		registered_at TIMESTAMP WITH TIME ZONE
	);`
	if _, err := db.Exec(userTableSQL); err != nil {
		log.Fatalf("Failed to create users table: %v", err)
	}
	log.Println("Auth Service tables created successfully.")
}

// --- Models ---
// InternalUser represents a user in our platform with a unique tenant ID.
type InternalUser struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenantId"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	RegisteredAt time.Time `json:"registeredAt"`
}

// UserSocialAccount is a model for data sent to the Account Service.
type UserSocialAccount struct {
	UserID         string    `json:"userId"`
	TenantID       string    `json:"tenantId"`
	Platform       string    `json:"platform"`
	PlatformUserID string    `json:"platformUserId"`
	AccessToken    string    `json:"accessToken"`
	RefreshToken   string    `json:"refreshToken"`
	ExpiresAt      time.Time `json:"expiresAt"`
	Username       string    `json:"username"`
	ProfilePic     string    `json:"profilePic"`
}

// Claims for our JWT, including the TenantID.
type Claims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	jwt.RegisteredClaims
}

// --- Database Operations ---
// saveInternalUser inserts or updates a user in the database.
func saveInternalUser(user InternalUser) error {
	_, err := db.Exec(
		"INSERT INTO users (id, tenant_id, email, name, registered_at) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id) DO UPDATE SET tenant_id = $2, email = $3, name = $4",
		user.ID, user.TenantID, user.Email, user.Name, user.RegisteredAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}
	return nil
}

// getInternalUserByEmail retrieves a user by email from the database.
func getInternalUserByEmail(email string) (InternalUser, bool, error) {
	var user InternalUser
	row := db.QueryRow("SELECT id, tenant_id, email, name, registered_at FROM users WHERE email = $1", email)
	err := row.Scan(&user.ID, &user.TenantID, &user.Email, &user.Name, &user.RegisteredAt)
	if err == sql.ErrNoRows {
		return user, false, nil
	}
	if err != nil {
		return user, false, fmt.Errorf("failed to get user by email: %w", err)
	}
	return user, true, nil
}

// --- JWT Helper ---
// generateJWT creates a new JWT with the UserID and TenantID claims.
func generateJWT(userID, tenantID string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID:   userID,
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(JWT_SECRET))
	if err != nil {
		return "", fmt.Errorf("could not sign token: %w", err)
	}
	return tokenString, nil
}

// --- OAuth Handlers ---
// handleMetaLogin redirects the user to the Meta authorization page.
func handleMetaLogin(w http.ResponseWriter, r *http.Request) {
	scope := "email,public_profile,pages_show_list,instagram_basic"
	authURL := fmt.Sprintf(
		"https://www.facebook.com/v19.0/dialog/oauth?client_id=%s&redirect_uri=%s&scope=%s&response_type=code",
		META_CLIENT_ID, url.QueryEscape(META_REDIRECT_URI), url.QueryEscape(scope),
	)
	http.Redirect(w, r, authURL, http.StatusFound)
}

// handleMetaCallback handles the redirect from Meta and completes the OAuth flow.
func handleMetaCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Authorization code not found", http.StatusBadRequest)
		return
	}
	tokenURL := fmt.Sprintf("https://graph.facebook.com/v19.0/oauth/access_token?client_id=%s&redirect_uri=%s&client_secret=%s&code=%s", META_CLIENT_ID, url.QueryEscape(META_REDIRECT_URI), META_CLIENT_SECRET, code)
	tokenResp, err := http.Get(tokenURL)
	if err != nil {
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(tokenResp.Body)
		log.Printf("Token exchange failed with status %d: %s", tokenResp.StatusCode, string(body))
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}
	var tokenData map[string]interface{}
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil {
		http.Error(w, "Failed to parse token data", http.StatusInternalServerError)
		return
	}
	accessToken, ok := tokenData["access_token"].(string)
	if !ok {
		http.Error(w, "Access token missing", http.StatusInternalServerError)
		return
	}
	profileURL := fmt.Sprintf("https://graph.facebook.com/v19.0/me?fields=id,name,email,picture&access_token=%s", accessToken)
	profileResp, err := http.Get(profileURL)
	if err != nil {
		http.Error(w, "Failed to fetch user profile", http.StatusInternalServerError)
		return
	}
	defer profileResp.Body.Close()
	if profileResp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(profileResp.Body)
		log.Printf("Profile fetch failed with status %d: %s", profileResp.StatusCode, string(body))
		http.Error(w, "Failed to fetch user profile", http.StatusInternalServerError)
		return
	}
	var profileData map[string]interface{}
	if err := json.NewDecoder(profileResp.Body).Decode(&profileData); err != nil {
		http.Error(w, "Failed to parse profile data", http.StatusInternalServerError)
		return
	}
	platformUserID, ok := profileData["id"].(string)
	if !ok {
		http.Error(w, "Meta User ID not found", http.StatusInternalServerError)
		return
	}
	userName, _ := profileData["name"].(string)
	userEmail, _ := profileData["email"].(string)
	pictureData, _ := profileData["picture"].(map[string]interface{})
	dataData, _ := pictureData["data"].(map[string]interface{})
	profilePicURL, _ := dataData["url"].(string)

	var currentUser InternalUser
	user, found, err := getInternalUserByEmail(userEmail)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if found {
		currentUser = user
	} else {
		newTenantID := uuid.New().String()
		currentUser = InternalUser{
			ID:           platformUserID,
			TenantID:     newTenantID,
			Email:        userEmail,
			Name:         userName,
			RegisteredAt: time.Now(),
		}
		if err := saveInternalUser(currentUser); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
	
	linkedAccount := UserSocialAccount{
		UserID:         currentUser.ID,
		TenantID:       currentUser.TenantID,
		Platform:       "Meta",
		PlatformUserID: platformUserID,
		AccessToken:    accessToken,
		ExpiresAt:      time.Now().Add(60 * time.Days),
		Username:       userName,
		ProfilePic:     profilePicURL,
	}

	jsonPayload, _ := json.Marshal(linkedAccount)
	resp, err := http.Post(fmt.Sprintf("%s/accounts", ACCOUNT_SERVICE_URL), "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil || resp.StatusCode != http.StatusCreated {
		log.Printf("Failed to call Account Service: %v", err)
		http.Error(w, "Failed to link social account", http.StatusInternalServerError)
		return
	}

	jwtToken, err := generateJWT(currentUser.ID, currentUser.TenantID)
	if err != nil {
		http.Error(w, "Failed to generate session token", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "http://localhost:3000/auth-success?token="+jwtToken, http.StatusFound)
}

// handleTikTokLogin redirects the user to the TikTok authorization page.
func handleTikTokLogin(w http.ResponseWriter, r *http.Request) {
	scope := "user.info.basic,video.list,video.upload"
	authURL := fmt.Sprintf(
		"https://www.tiktok.com/v2/auth/authorize?client_key=%s&redirect_uri=%s&scope=%s&response_type=code",
		TIKTOK_CLIENT_KEY, url.QueryEscape(TIKTOK_REDIRECT_URI), url.QueryEscape(scope),
	)
	http.Redirect(w, r, authURL, http.StatusFound)
}

// handleTikTokCallback handles the redirect from TikTok and completes the OAuth flow.
func handleTikTokCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Authorization code not found", http.StatusBadRequest)
		return
	}

	tokenURL := "https://open-api.tiktok.com/oauth/access_token/"
	payload := fmt.Sprintf(
		`{"client_key": "%s", "client_secret": "%s", "code": "%s", "grant_type": "authorization_code"}`,
		TIKTOK_CLIENT_KEY, TIKTOK_CLIENT_SECRET, code,
	)
	
	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(payload))
	if err != nil {
		http.Error(w, "Failed to create token request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	tokenResp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(tokenResp.Body)
		log.Printf("TikTok token exchange failed with status %d: %s", tokenResp.StatusCode, string(body))
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

	var tokenData map[string]interface{}
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil {
		http.Error(w, "Failed to parse token data", http.StatusInternalServerError)
		return
	}

	accessToken, _ := tokenData["access_token"].(string)
	refreshToken, _ := tokenData["refresh_token"].(string)
	expiresIn, _ := tokenData["expires_in"].(float64)
	openID, _ := tokenData["open_id"].(string)
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)
	
	userName := "TikTokUser"
	profilePicURL := "https://placehold.co/100x100/FF0050/FFFFFF?text=T"
	
	newTenantID := uuid.New().String()
	currentUser := InternalUser{
		ID: openID,
		TenantID: newTenantID,
		Email: openID + "@tiktok.com",
		Name: "TikTok User",
		RegisteredAt: time.Now(),
	}
	
	linkedAccount := UserSocialAccount{
		UserID:         currentUser.ID,
		TenantID:       currentUser.TenantID,
		Platform:       "TikTok",
		PlatformUserID: openID,
		AccessToken:    accessToken,
		RefreshToken:   refreshToken,
		ExpiresAt:      expiresAt,
		Username:       userName,
		ProfilePic:     profilePicURL,
	}

	jsonPayload, _ := json.Marshal(linkedAccount)
	resp, err := http.Post(fmt.Sprintf("%s/accounts", ACCOUNT_SERVICE_URL), "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil || resp.StatusCode != http.StatusCreated {
		log.Printf("Failed to call Account Service: %v", err)
		http.Error(w, "Failed to link social account", http.StatusInternalServerError)
		return
	}
	
	jwtToken, err := generateJWT(currentUser.ID, currentUser.TenantID)
	if err != nil {
		http.Error(w, "Failed to generate session token", http.StatusInternalServerError)
		return
	}
	
	http.Redirect(w, r, "http://localhost:3000/auth-success?token="+jwtToken, http.StatusFound)
}

// handleSnapchatLogin redirects the user to the Snapchat authorization page.
func handleSnapchatLogin(w http.ResponseWriter, r *http.Request) {
	scope := "snapchat-ads.manage,snapchat-creative-kit.creative-kit-token"
	authURL := fmt.Sprintf(
		"https://accounts.snapchat.com/login/oauth2/authorize?client_id=%s&redirect_uri=%s&scope=%s&response_type=code",
		SNAPCHAT_CLIENT_ID, url.QueryEscape(SNAPCHAT_REDIRECT_URI), url.QueryEscape(scope),
	)
	http.Redirect(w, r, authURL, http.StatusFound)
}

// handleSnapchatCallback handles the redirect from Snapchat and completes the OAuth flow.
func handleSnapchatCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Authorization code not found", http.StatusBadRequest)
		return
	}
	
	tokenURL := "https://accounts.snapchat.com/login/oauth2/access_token"
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", SNAPCHAT_REDIRECT_URI)
	data.Set("client_id", SNAPCHAT_CLIENT_ID)
	data.Set("client_secret", SNAPCHAT_CLIENT_SECRET)
	
	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		http.Error(w, "Failed to create token request", http.StatusInternalServerError)
		return
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	tokenResp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(tokenResp.Body)
		log.Printf("Snapchat token exchange failed with status %d: %s", tokenResp.StatusCode, string(body))
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

	var tokenData map[string]interface{}
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil {
		http.Error(w, "Failed to parse token data", http.StatusInternalServerError)
		return
	}

	accessToken, _ := tokenData["access_token"].(string)
	refreshToken, _ := tokenData["refresh_token"].(string)
	expiresIn, _ := tokenData["expires_in"].(float64)
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)
	
	platformUserID := "SnapchatUser"
	userName := "Snapchat User"
	profilePicURL := "https://placehold.co/100x100/FFFC00/000000?text=S"
	
	newTenantID := uuid.New().String()
	currentUser := InternalUser{
		ID:           platformUserID,
		TenantID:     newTenantID,
		Email:        platformUserID + "@snapchat.com",
		Name:         "Snapchat User",
		RegisteredAt: time.Now(),
	}
	
	linkedAccount := UserSocialAccount{
		UserID:         currentUser.ID,
		TenantID:       currentUser.TenantID,
		Platform:       "Snapchat",
		PlatformUserID: platformUserID,
		AccessToken:    accessToken,
		RefreshToken:   refreshToken,
		ExpiresAt:      expiresAt,
		Username:       userName,
		ProfilePic:     profilePicURL,
	}
	jsonPayload, _ := json.Marshal(linkedAccount)
	resp, err := http.Post(fmt.Sprintf("%s/accounts", ACCOUNT_SERVICE_URL), "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil || resp.StatusCode != http.StatusCreated {
		log.Printf("Failed to call Account Service: %v", err)
		http.Error(w, "Failed to link social account", http.StatusInternalServerError)
		return
	}
	
	jwtToken, err := generateJWT(currentUser.ID, currentUser.TenantID)
	if err != nil {
		http.Error(w, "Failed to generate session token", http.StatusInternalServerError)
		return
	}
	
	http.Redirect(w, r, "http://localhost:3000/auth-success?token="+jwtToken, http.StatusFound)
}

// --- Main function ---

func main() {
	initDB()
	defer db.Close()

	router := mux.NewRouter()

	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Access-Control-Allow-Headers, Authorization, X-Requested-With")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	router.HandleFunc("/oauth/meta/login", handleMetaLogin).Methods("GET")
	router.HandleFunc("/oauth/meta/callback", handleMetaCallback).Methods("GET")
	router.HandleFunc("/oauth/tiktok/login", handleTikTokLogin).Methods("GET")
	router.HandleFunc("/oauth/tiktok/callback", handleTikTokCallback).Methods("GET")
	router.HandleFunc("/oauth/snapchat/login", handleSnapchatLogin).Methods("GET")
	router.HandleFunc("/oauth/snapchat/callback", handleSnapchatCallback).Methods("GET")
	
	log.Println("Auth Service is starting on port 8081...")
	log.Fatal(http.ListenAndServe(":8081", router))
}

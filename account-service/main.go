package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// --- Configuration ---
const (
	JWT_SECRET = "supersecretjwtkeythatshouldbeverylongandrandom"
)

// --- Database Connection ---
var db *sql.DB

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
	log.Println("Account Service successfully connected to PostgreSQL!")
	createTables()
}

func createTables() {
	socialAccountTableSQL := `
	CREATE TABLE IF NOT EXISTS social_accounts (
		user_id TEXT,
		tenant_id TEXT NOT NULL,
		platform TEXT NOT NULL,
		platform_user_id TEXT PRIMARY KEY,
		access_token TEXT NOT NULL,
		refresh_token TEXT,
		expires_at TIMESTAMP WITH TIME ZONE,
		username TEXT,
		profile_pic TEXT
	);`
	if _, err := db.Exec(socialAccountTableSQL); err != nil {
		log.Fatalf("Failed to create social_accounts table: %v", err)
	}
	log.Println("Account Service tables created successfully.")
}

// --- Models ---
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

type Claims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	jwt.RegisteredClaims
}

// --- Database Operations ---
func saveUserSocialAccount(account UserSocialAccount) error {
	_, err := db.Exec(
		"INSERT INTO social_accounts (user_id, tenant_id, platform, platform_user_id, access_token, refresh_token, expires_at, username, profile_pic) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) ON CONFLICT (platform_user_id) DO UPDATE SET user_id = $1, tenant_id = $2, access_token = $5, expires_at = $7, username = $8, profile_pic = $9",
		account.UserID, account.TenantID, account.Platform, account.PlatformUserID, account.AccessToken, account.RefreshToken, account.ExpiresAt, account.Username, account.ProfilePic,
	)
	if err != nil {
		return fmt.Errorf("failed to save social account: %w", err)
	}
	return nil
}

func getSocialAccountsForUser(userID, tenantID string) ([]UserSocialAccount, error) {
	rows, err := db.Query("SELECT user_id, tenant_id, platform, platform_user_id, access_token, refresh_token, expires_at, username, profile_pic FROM social_accounts WHERE user_id = $1 AND tenant_id = $2", userID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get social accounts: %w", err)
	}
	defer rows.Close()

	var accounts []UserSocialAccount
	for rows.Next() {
		var account UserSocialAccount
		var refreshToken sql.NullString
		if err := rows.Scan(&account.UserID, &account.TenantID, &account.Platform, &account.PlatformUserID, &account.AccessToken, &refreshToken, &account.ExpiresAt, &account.Username, &account.ProfilePic); err != nil {
			return nil, fmt.Errorf("failed to scan social account row: %w", err)
		}
		if refreshToken.Valid {
			account.RefreshToken = refreshToken.String
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}

// --- Middleware ---
type contextKey string
const userIDKey contextKey = "userID"
const tenantIDKey contextKey = "tenantID"

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}
		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(JWT_SECRET), nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
		ctx = context.WithValue(ctx, tenantIDKey, claims.TenantID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getUserIDAndTenantIDFromContext(ctx context.Context) (string, string, error) {
	userID, ok := ctx.Value(userIDKey).(string)
	if !ok {
		return "", "", fmt.Errorf("user ID not found in context")
	}
	tenantID, ok := ctx.Value(tenantIDKey).(string)
	if !ok {
		return "", "", fmt.Errorf("tenant ID not found in context")
	}
	return userID, tenantID, nil
}

// --- Handlers ---
func createAccountHandler(w http.ResponseWriter, r *http.Request) {
	var newAccount UserSocialAccount
	if err := json.NewDecoder(r.Body).Decode(&newAccount); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := saveUserSocialAccount(newAccount); err != nil {
		log.Printf("Failed to save social account: %v", err)
		http.Error(w, "Failed to save social account", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newAccount)
}

func getAccountsHandler(w http.ResponseWriter, r *http.Request) {
	userID, tenantID, err := getUserIDAndTenantIDFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	accounts, err := getSocialAccountsForUser(userID, tenantID)
	if err != nil {
		http.Error(w, "Failed to retrieve accounts", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(accounts)
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
	
	router.HandleFunc("/accounts", createAccountHandler).Methods("POST")
	
	apiRouter := router.PathPrefix("/api").Subrouter()
	apiRouter.Use(authMiddleware)
	apiRouter.HandleFunc("/accounts", getAccountsHandler).Methods("GET")
	
	log.Println("Account Service is starting on port 8082...")
	log.Fatal(http.ListenAndServe(":8082", router))
}

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
	log.Println("Post Service successfully connected to PostgreSQL!")
	createTables()
}

func createTables() {
	postTableSQL := `
	CREATE TABLE IF NOT EXISTS posts (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		tenant_id TEXT NOT NULL,
		platform TEXT NOT NULL,
		content TEXT NOT NULL,
		media_url TEXT,
		scheduled_at TIMESTAMP WITH TIME ZONE NOT NULL,
		posted_at TIMESTAMP WITH TIME ZONE,
		status TEXT NOT NULL
	);`
	if _, err := db.Exec(postTableSQL); err != nil {
		log.Fatalf("Failed to create posts table: %v", err)
	}
	log.Println("Post Service tables created successfully.")
}

// --- Models ---
type Post struct {
	ID            string    `json:"id"`
	UserID        string    `json:"userId"`
	TenantID      string    `json:"tenantId"`
	Platform      string    `json:"platform"`
	Content       string    `json:"content"`
	MediaURL      string    `json:"mediaUrl"`
	ScheduledAt   time.Time `json:"scheduledAt"`
	PostedAt      *time.Time `json:"postedAt"`
	Status        string    `json:"status"`
}

type Claims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	jwt.RegisteredClaims
}

// --- Database Operations ---
func savePost(post Post) error {
	_, err := db.Exec(
		"INSERT INTO posts (id, user_id, tenant_id, platform, content, media_url, scheduled_at, status) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		post.ID, post.UserID, post.TenantID, post.Platform, post.Content, post.MediaURL, post.ScheduledAt, post.Status,
	)
	if err != nil {
		return fmt.Errorf("failed to save post: %w", err)
	}
	return nil
}

func getPostsForUser(userID, tenantID string) ([]Post, error) {
	rows, err := db.Query("SELECT id, user_id, tenant_id, platform, content, media_url, scheduled_at, posted_at, status FROM posts WHERE user_id = $1 AND tenant_id = $2 ORDER BY scheduled_at DESC", userID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get posts: %w", err)
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		var postedAt sql.NullTime
		if err := rows.Scan(&post.ID, &post.UserID, &post.TenantID, &post.Platform, &post.Content, &post.MediaURL, &post.ScheduledAt, &postedAt, &post.Status); err != nil {
			return nil, fmt.Errorf("failed to scan post row: %w", err)
		}
		if postedAt.Valid {
			post.PostedAt = &postedAt.Time
		}
		posts = append(posts, post)
	}
	return posts, nil
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
func getScheduledPostsHandler(w http.ResponseWriter, r *http.Request) {
	userID, tenantID, err := getUserIDAndTenantIDFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	posts, err := getPostsForUser(userID, tenantID)
	if err != nil {
		http.Error(w, "Failed to retrieve posts", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}

func createPostHandler(w http.ResponseWriter, r *http.Request) {
	userID, tenantID, err := getUserIDAndTenantIDFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var newPost Post
	if err := json.NewDecoder(r.Body).Decode(&newPost); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	newPost.ID = fmt.Sprintf("post-%d", time.Now().UnixNano())
	newPost.UserID = userID
	newPost.TenantID = tenantID
	newPost.Status = "scheduled"
	if err := savePost(newPost); err != nil {
		http.Error(w, "Failed to save post", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newPost)
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

	apiRouter := router.PathPrefix("/api").Subrouter()
	apiRouter.Use(authMiddleware)

	apiRouter.HandleFunc("/posts", getScheduledPostsHandler).Methods("GET")
	apiRouter.HandleFunc("/posts", createPostHandler).Methods("POST")
	
	log.Println("Post Service is starting on port 8083...")
	log.Fatal(http.ListenAndServe(":8083", router))
}

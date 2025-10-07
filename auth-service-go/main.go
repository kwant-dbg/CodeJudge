package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"codejudge/common/dbutil"
	"codejudge/common/env"
	"codejudge/common/health"
	"codejudge/common/httpx"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var (
	logger    *zap.Logger
	dbManager *dbutil.ConnectionManager
	jwtSecret []byte
)

// User represents a user in the system
type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // Never serialize password
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AuthRequest represents login/register request
type AuthRequest struct {
	Username string `json:"username"`
	Email    string `json:"email,omitempty"`
	Password string `json:"password"`
}

// AuthResponse represents authentication response
type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// Claims represents JWT claims
type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func init() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
}

func connectDB() {
	databaseURL := env.Get("DATABASE_URL", "")
	if databaseURL == "" {
		logger.Fatal("DATABASE_URL not set")
	}
	dbManager = dbutil.ConnectManagerWithRetry(logger, databaseURL, 5, 2*time.Second)
	createTables()
}

func initJWTSecret() {
	// In production, use environment variable or Azure Key Vault
	secretEnv := env.Get("JWT_SECRET", "")
	if secretEnv != "" {
		jwtSecret = []byte(secretEnv)
	} else {
		// Generate random secret for development
		jwtSecret = make([]byte, 32)
		if _, err := rand.Read(jwtSecret); err != nil {
			logger.Fatal("Failed to generate JWT secret", zap.Error(err))
		}
		logger.Info("Generated random JWT secret for development")
	}
}

func createTables() {
	createUsersTable := `
    CREATE TABLE IF NOT EXISTS users (
        id SERIAL PRIMARY KEY,
        username VARCHAR(50) UNIQUE NOT NULL,
        email VARCHAR(255) UNIQUE NOT NULL,
        password_hash VARCHAR(255) NOT NULL,
        role VARCHAR(20) DEFAULT 'user',
        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
    );

    -- Add indexes for performance
    CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
    CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
    
    -- Add trigger to update updated_at column
    CREATE OR REPLACE FUNCTION update_updated_at_column()
    RETURNS TRIGGER AS $$
    BEGIN
        NEW.updated_at = CURRENT_TIMESTAMP;
        RETURN NEW;
    END;
    $$ language 'plpgsql';

    DROP TRIGGER IF EXISTS update_users_updated_at ON users;
    CREATE TRIGGER update_users_updated_at
        BEFORE UPDATE ON users
        FOR EACH ROW
        EXECUTE FUNCTION update_updated_at_column();
    `

	if _, err := dbManager.GetDB().Exec(createUsersTable); err != nil {
		logger.Fatal("Failed to create users table", zap.Error(err))
	}
	logger.Info("Users table is ready")
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func checkPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func generateToken(user User) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour) // 24 hours
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   fmt.Sprintf("%d", user.ID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, "Invalid request body", http.StatusBadRequest, err, logger)
		return
	}

	// Validate input
	if req.Username == "" || req.Email == "" || req.Password == "" {
		httpx.Error(w, "Username, email, and password are required", http.StatusBadRequest, nil, logger)
		return
	}

	if len(req.Password) < 6 {
		httpx.Error(w, "Password must be at least 6 characters", http.StatusBadRequest, nil, logger)
		return
	}

	// Hash password
	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		httpx.Error(w, "Failed to process password", http.StatusInternalServerError, err, logger)
		return
	}

	// Insert user
	var user User
	query := `
        INSERT INTO users (username, email, password_hash, role) 
        VALUES ($1, $2, $3, 'user') 
        RETURNING id, username, email, role, created_at, updated_at`

	err = dbManager.GetDB().QueryRowContext(r.Context(), query, req.Username, req.Email, hashedPassword).
		Scan(&user.ID, &user.Username, &user.Email, &user.Role, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			httpx.Error(w, "Username or email already exists", http.StatusConflict, err, logger)
		} else {
			httpx.Error(w, "Failed to create user", http.StatusInternalServerError, err, logger)
		}
		return
	}

	// Generate token
	token, err := generateToken(user)
	if err != nil {
		httpx.Error(w, "Failed to generate token", http.StatusInternalServerError, err, logger)
		return
	}

	response := AuthResponse{
		Token: token,
		User:  user,
	}

	logger.Info("User registered successfully", zap.String("username", user.Username), zap.Int("user_id", user.ID))
	httpx.JSON(w, http.StatusCreated, response)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, "Invalid request body", http.StatusBadRequest, err, logger)
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		httpx.Error(w, "Username and password are required", http.StatusBadRequest, nil, logger)
		return
	}

	// Find user
	var user User
	var passwordHash string
	query := `
        SELECT id, username, email, password_hash, role, created_at, updated_at 
        FROM users 
        WHERE username = $1`

	err := dbManager.GetDB().QueryRowContext(r.Context(), query, req.Username).
		Scan(&user.ID, &user.Username, &user.Email, &passwordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			httpx.Error(w, "Invalid username or password", http.StatusUnauthorized, nil, logger)
		} else {
			httpx.Error(w, "Database error", http.StatusInternalServerError, err, logger)
		}
		return
	}

	// Check password
	if !checkPassword(req.Password, passwordHash) {
		httpx.Error(w, "Invalid username or password", http.StatusUnauthorized, nil, logger)
		return
	}

	// Generate token
	token, err := generateToken(user)
	if err != nil {
		httpx.Error(w, "Failed to generate token", http.StatusInternalServerError, err, logger)
		return
	}

	response := AuthResponse{
		Token: token,
		User:  user,
	}

	logger.Info("User logged in successfully", zap.String("username", user.Username), zap.Int("user_id", user.ID))
	httpx.JSON(w, http.StatusOK, response)
}

func meHandler(w http.ResponseWriter, r *http.Request) {
	// This endpoint requires authentication - will be handled by middleware
	userID := r.Context().Value("user_id").(int)

	var user User
	query := `
        SELECT id, username, email, role, created_at, updated_at 
        FROM users 
        WHERE id = $1`

	err := dbManager.GetDB().QueryRowContext(r.Context(), query, userID).
		Scan(&user.ID, &user.Username, &user.Email, &user.Role, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			httpx.Error(w, "User not found", http.StatusNotFound, nil, logger)
		} else {
			httpx.Error(w, "Database error", http.StatusInternalServerError, err, logger)
		}
		return
	}

	httpx.JSON(w, http.StatusOK, user)
}

func validateTokenHandler(w http.ResponseWriter, r *http.Request) {
	// Extract token from header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		httpx.Error(w, "Authorization header required", http.StatusUnauthorized, nil, logger)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		httpx.Error(w, "Bearer token required", http.StatusUnauthorized, nil, logger)
		return
	}

	// Parse and validate token
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		httpx.Error(w, "Invalid token", http.StatusUnauthorized, err, logger)
		return
	}

	// Return claims
	httpx.JSON(w, http.StatusOK, claims)
}

func setupRoutes() *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // In production, specify your frontend domain
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Public routes
	r.Post("/register", registerHandler)
	r.Post("/login", loginHandler)
	r.Post("/validate", validateTokenHandler)

	// Protected routes (will need auth middleware in other services)
	r.Get("/me", meHandler) // This will need auth middleware

	// Health check
	r.Get("/health", health.HealthHandler())

	return r
}

func main() {
	defer logger.Sync()

	connectDB()
	initJWTSecret()

	router := setupRoutes()
	port := env.Get("PORT", "8003")

	logger.Info("Auth service starting", zap.String("port", port))
	if err := http.ListenAndServe(":"+port, router); err != nil {
		logger.Fatal("Server failed to start", zap.Error(err))
	}
}

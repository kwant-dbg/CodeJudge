package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	commonauth "codejudge/common/auth"
	"codejudge/common/dbutil"
	"codejudge/common/httpx"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
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

type AuthHandler struct {
	logger    *zap.Logger
	dbManager *dbutil.ConnectionManager
	jwtSecret []byte
}

func NewAuthHandler(logger *zap.Logger, dbManager *dbutil.ConnectionManager, jwtSecret []byte) *AuthHandler {
	return &AuthHandler{
		logger:    logger,
		dbManager: dbManager,
		jwtSecret: jwtSecret,
	}
}

func (h *AuthHandler) CreateTables() {
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

	if _, err := h.dbManager.GetDB().Exec(createUsersTable); err != nil {
		h.logger.Fatal("Failed to create users table", zap.Error(err))
	}
	h.logger.Info("Users table is ready")
}

func (h *AuthHandler) hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func (h *AuthHandler) checkPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (h *AuthHandler) generateToken(user User) (string, error) {
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
	return token.SignedString(h.jwtSecret)
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate input
	if req.Username == "" || req.Email == "" || req.Password == "" {
		httpx.Error(w, http.StatusBadRequest, "Username, email, and password are required")
		return
	}

	if len(req.Password) < 6 {
		httpx.Error(w, http.StatusBadRequest, "Password must be at least 6 characters")
		return
	}

	// Hash password
	hashedPassword, err := h.hashPassword(req.Password)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "Failed to process password")
		return
	}

	// Insert user
	var user User
	query := `
        INSERT INTO users (username, email, password_hash, role) 
        VALUES ($1, $2, $3, 'user') 
        RETURNING id, username, email, role, created_at, updated_at`

	err = h.dbManager.GetDB().QueryRowContext(r.Context(), query, req.Username, req.Email, hashedPassword).
		Scan(&user.ID, &user.Username, &user.Email, &user.Role, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			httpx.Error(w, http.StatusConflict, "Username or email already exists")
		} else {
			httpx.Error(w, http.StatusInternalServerError, "Failed to create user")
		}
		return
	}

	// Generate token
	token, err := h.generateToken(user)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	response := AuthResponse{
		Token: token,
		User:  user,
	}

	h.logger.Info("User registered successfully", zap.String("username", user.Username), zap.Int("user_id", user.ID))
	httpx.JSON(w, http.StatusCreated, response)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		httpx.Error(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	// Find user
	var user User
	var passwordHash string
	query := `
        SELECT id, username, email, password_hash, role, created_at, updated_at 
        FROM users 
        WHERE username = $1`

	err := h.dbManager.GetDB().QueryRowContext(r.Context(), query, req.Username).
		Scan(&user.ID, &user.Username, &user.Email, &passwordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			httpx.Error(w, http.StatusUnauthorized, "Invalid username or password")
		} else {
			httpx.Error(w, http.StatusInternalServerError, "Database error")
		}
		return
	}

	// Check password
	if !h.checkPassword(req.Password, passwordHash) {
		httpx.Error(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	// Generate token
	token, err := h.generateToken(user)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	response := AuthResponse{
		Token: token,
		User:  user,
	}

	h.logger.Info("User logged in successfully", zap.String("username", user.Username), zap.Int("user_id", user.ID))
	httpx.JSON(w, http.StatusOK, response)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := commonauth.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var user User
	query := `
        SELECT id, username, email, role, created_at, updated_at 
        FROM users 
        WHERE id = $1`

	err := h.dbManager.GetDB().QueryRowContext(r.Context(), query, userID).
		Scan(&user.ID, &user.Username, &user.Email, &user.Role, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			httpx.Error(w, http.StatusNotFound, "User not found")
		} else {
			httpx.Error(w, http.StatusInternalServerError, "Database error")
		}
		return
	}

	httpx.JSON(w, http.StatusOK, user)
}

func (h *AuthHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	// Extract token from header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		httpx.Error(w, http.StatusUnauthorized, "Authorization header required")
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		httpx.Error(w, http.StatusUnauthorized, "Bearer token required")
		return
	}

	// Parse and validate token
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return h.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		httpx.Error(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	// Return claims
	httpx.JSON(w, http.StatusOK, claims)
}
package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"codejudge/common/httpx"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// UserContext represents the authenticated user information
type UserContext struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// Claims represents JWT claims structure
type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret       []byte
	TokenExpiration time.Duration
	RequiredRoles   []string // Optional: restrict to specific roles
	AllowAnonymous  bool     // If true, continues without auth but doesn't set user context
	AuthServiceURL  string   // URL to validate tokens with auth service
	SkipPaths       []string // Paths that don't require authentication
}

// DefaultAuthConfig returns sensible defaults
func DefaultAuthConfig(jwtSecret []byte) AuthConfig {
	return AuthConfig{
		JWTSecret:       jwtSecret,
		TokenExpiration: 24 * time.Hour,
		RequiredRoles:   []string{"user", "admin"},
		AllowAnonymous:  false,
		SkipPaths:       []string{"/health", "/ready"},
	}
}

// JWTMiddleware creates a middleware for JWT token validation
func JWTMiddleware(config AuthConfig, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path should skip authentication
			for _, skipPath := range config.SkipPaths {
				if r.URL.Path == skipPath {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				if config.AllowAnonymous {
					next.ServeHTTP(w, r)
					return
				}
				httpx.Error(w, http.StatusUnauthorized, "Authorization header required")
				return
			}

			// Check Bearer prefix
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				httpx.Error(w, http.StatusUnauthorized, "Bearer token required")
				return
			}

			// Parse and validate token
			userCtx, err := ValidateToken(tokenString, config.JWTSecret)
			if err != nil {
				httpx.Error(w, http.StatusUnauthorized, "Invalid token")
				return
			}

			// Check required roles if specified
			if len(config.RequiredRoles) > 0 {
				hasValidRole := false
				for _, role := range config.RequiredRoles {
					if userCtx.Role == role {
						hasValidRole = true
						break
					}
				}
				if !hasValidRole {
					httpx.Error(w, http.StatusForbidden, "Insufficient permissions")
					return
				}
			}

			// Add user context to request
			ctx := context.WithValue(r.Context(), "user", userCtx)
			ctx = context.WithValue(ctx, "user_id", userCtx.UserID)
			ctx = context.WithValue(ctx, "username", userCtx.Username)
			ctx = context.WithValue(ctx, "role", userCtx.Role)

			// Continue with authenticated request
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ValidateToken parses and validates a JWT token
func ValidateToken(tokenString string, jwtSecret []byte) (*UserContext, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Check expiration
	if claims.RegisteredClaims.ExpiresAt != nil && claims.RegisteredClaims.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("token expired")
	}

	return &UserContext{
		UserID:   claims.UserID,
		Username: claims.Username,
		Role:     claims.Role,
	}, nil
}

// GetUserFromContext extracts user context from request context
func GetUserFromContext(ctx context.Context) (*UserContext, bool) {
	user, ok := ctx.Value("user").(*UserContext)
	return user, ok
}

// GetUserIDFromContext extracts user ID from request context
func GetUserIDFromContext(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value("user_id").(int)
	return userID, ok
}

// RequireAuth is a helper middleware that requires authentication
func RequireAuth(jwtSecret []byte, logger *zap.Logger) func(http.Handler) http.Handler {
	config := DefaultAuthConfig(jwtSecret)
	return JWTMiddleware(config, logger)
}

// RequireRole creates middleware that requires specific role
func RequireRole(jwtSecret []byte, roles []string, logger *zap.Logger) func(http.Handler) http.Handler {
	config := DefaultAuthConfig(jwtSecret)
	config.RequiredRoles = roles
	return JWTMiddleware(config, logger)
}

// OptionalAuth allows anonymous access but sets user context if token provided
func OptionalAuth(jwtSecret []byte, logger *zap.Logger) func(http.Handler) http.Handler {
	config := DefaultAuthConfig(jwtSecret)
	config.AllowAnonymous = true
	return JWTMiddleware(config, logger)
}

package httpx

import (
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	Error     string    `json:"error"`
	Code      string    `json:"code,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	RequestID string    `json:"request_id,omitempty"`
}

// ServiceError represents an internal service error
type ServiceError struct {
	Message   string
	Code      string
	HTTPCode  int
	Cause     error
	RequestID string
}

func (e *ServiceError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// NewServiceError creates a new service error
func NewServiceError(message, code string, httpCode int, cause error) *ServiceError {
	return &ServiceError{
		Message:  message,
		Code:     code,
		HTTPCode: httpCode,
		Cause:    cause,
	}
}

func JSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, map[string]string{"error": msg})
}

// ErrorWithDetails sends a structured error response
func ErrorWithDetails(w http.ResponseWriter, err *ServiceError, logger *zap.Logger) {
	requestID := err.RequestID
	if requestID == "" {
		// Try to get request ID from context if available
		if reqID := w.Header().Get("X-Request-ID"); reqID != "" {
			requestID = reqID
		}
	}

	response := ErrorResponse{
		Error:     err.Message,
		Code:      err.Code,
		Timestamp: time.Now().UTC(),
		RequestID: requestID,
	}

	// Log the error with context
	if logger != nil {
		logFields := []zap.Field{
			zap.String("error_code", err.Code),
			zap.String("request_id", requestID),
			zap.Int("http_status", err.HTTPCode),
		}
		if err.Cause != nil {
			logFields = append(logFields, zap.Error(err.Cause))
		}
		logger.Error(err.Message, logFields...)
	}

	JSON(w, err.HTTPCode, response)
}

// RecoveryMiddleware provides panic recovery
func RecoveryMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("Panic recovered",
						zap.Any("panic", err),
						zap.String("method", r.Method),
						zap.String("path", r.URL.Path),
					)

					serviceErr := NewServiceError(
						"Internal server error",
						"INTERNAL_ERROR",
						http.StatusInternalServerError,
						nil,
					)
					ErrorWithDetails(w, serviceErr, logger)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

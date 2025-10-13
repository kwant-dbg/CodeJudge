package health

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	HealthHandler()(rr, req)
	if rr.Code != 200 {
		t.Fatalf("expected 200 got %d", rr.Code)
	}
}

func TestReadyHandler_OK(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ready", nil)
	ReadyHandler(func(ctx context.Context) error { return nil })(rr, req)
	if rr.Code != 200 {
		t.Fatalf("expected 200 got %d", rr.Code)
	}
}

func TestReadyHandler_NotReady(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ready", nil)
	ReadyHandler(func(ctx context.Context) error { return errors.New("x") })(rr, req)
	if rr.Code != 503 {
		t.Fatalf("expected 503 got %d", rr.Code)
	}
}

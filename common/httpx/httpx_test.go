package httpx

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestJSON(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, 201, map[string]string{"ok": "1"})
	if w.Code != 201 {
		t.Fatalf("status %d", w.Code)
	}
	var m map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil || m["ok"] != "1" {
		t.Fatalf("bad body: %s", w.Body.String())
	}
}

func TestError(t *testing.T) {
	w := httptest.NewRecorder()
	Error(w, 400, "bad")
	if w.Code != 400 {
		t.Fatalf("status %d", w.Code)
	}
}

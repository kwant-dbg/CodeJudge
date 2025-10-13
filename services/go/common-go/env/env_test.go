package env

import (
	"os"
	"testing"
)

func TestGet_WithValue(t *testing.T) {
	const k = "ENV_TEST_KEY"
	os.Setenv(k, "v")
	defer os.Unsetenv(k)
	if got := Get(k, "d"); got != "v" {
		t.Fatalf("expected v, got %q", got)
	}
}

func TestGet_Default(t *testing.T) {
	const k = "ENV_TEST_KEY_MISSING"
	os.Unsetenv(k)
	if got := Get(k, "d"); got != "d" {
		t.Fatalf("expected default d, got %q", got)
	}
}

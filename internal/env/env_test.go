package env

import (
	"strings"
	"testing"
)

func TestGetString(t *testing.T) {
	t.Setenv("ENV_TEST_STRING", "configured")

	if got := GetString("ENV_TEST_STRING", "fallback"); got != "configured" {
		t.Fatalf("expected configured value, got %q", got)
	}
	if got := GetString("ENV_TEST_STRING_MISSING", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}
}

func TestGetInt(t *testing.T) {
	t.Setenv("ENV_TEST_INT", "42")

	got, err := GetInt("ENV_TEST_INT", 7)
	if err != nil {
		t.Fatalf("GetInt returned error: %v", err)
	}
	if got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}

	got, err = GetInt("ENV_TEST_INT_MISSING", 7)
	if err != nil {
		t.Fatalf("GetInt returned error for missing value: %v", err)
	}
	if got != 7 {
		t.Fatalf("expected fallback 7, got %d", got)
	}
}

func TestGetIntReportsParseError(t *testing.T) {
	t.Setenv("ENV_TEST_BAD_INT", "nope")

	_, err := GetInt("ENV_TEST_BAD_INT", 7)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parse ENV_TEST_BAD_INT as int") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetBool(t *testing.T) {
	t.Setenv("ENV_TEST_BOOL", "true")

	got, err := GetBool("ENV_TEST_BOOL", false)
	if err != nil {
		t.Fatalf("GetBool returned error: %v", err)
	}
	if !got {
		t.Fatal("expected true")
	}

	got, err = GetBool("ENV_TEST_BOOL_MISSING", true)
	if err != nil {
		t.Fatalf("GetBool returned error for missing value: %v", err)
	}
	if !got {
		t.Fatal("expected fallback true")
	}
}

func TestGetBoolReportsParseError(t *testing.T) {
	t.Setenv("ENV_TEST_BAD_BOOL", "maybe")

	_, err := GetBool("ENV_TEST_BAD_BOOL", false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parse ENV_TEST_BAD_BOOL as bool") {
		t.Fatalf("unexpected error: %v", err)
	}
}

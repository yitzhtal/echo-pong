package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestReadSecretFromFile(t *testing.T) {
	secretFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(secretFile, []byte("  test-secret\n"), 0o600); err != nil {
		t.Fatalf("write secret file: %v", err)
	}

	t.Setenv("SECRET_FILE_PATH", secretFile)
	if err := readSecretFromFile(); err != nil {
		t.Fatalf("readSecretFromFile returned error: %v", err)
	}

	if secret != "test-secret" {
		t.Fatalf("secret = %q, want %q", secret, "test-secret")
	}
}

func TestAuthMiddlewareAcceptsBearerToken(t *testing.T) {
	secret = "test-secret"

	called := false
	handler := authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Authorization", "Bearer test-secret")
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if !called {
		t.Fatal("expected wrapped handler to be called")
	}
}

func TestAuthMiddlewareRejectsInvalidToken(t *testing.T) {
	secret = "test-secret"

	handler := authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("wrapped handler should not be called")
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Authorization", "Bearer wrong-secret")
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestReadyHandler(t *testing.T) {
	ready.Store(false)
	rec := httptest.NewRecorder()
	readyHandler(rec, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status when not ready = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	ready.Store(true)
	t.Cleanup(func() {
		ready.Store(false)
	})

	rec = httptest.NewRecorder()
	readyHandler(rec, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status when ready = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestRequireMethodRejectsUnexpectedMethod(t *testing.T) {
	handler := requireMethod(http.MethodGet, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("wrapped handler should not be called")
	})

	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodPost, "/ping", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if allow := rec.Header().Get("Allow"); allow != http.MethodGet {
		t.Fatalf("Allow = %q, want %q", allow, http.MethodGet)
	}
}

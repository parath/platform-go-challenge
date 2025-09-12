package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/parath/platform-go-challenge/internal/favourites"
	"github.com/parath/platform-go-challenge/internal/httpapi"
)

// Smoke test: verify that the main server wiring works
func TestMainServerWiring(t *testing.T) {
	router := httpapi.NewServer(favourites.NewInMemoryStore())

	req := httptest.NewRequest(http.MethodGet, "/favourites/smoke", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// We expect either 200 (empty list) or 500 if store fails,
	// but never a panic or crash.
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for empty user, got %d", rr.Code)
	}
}

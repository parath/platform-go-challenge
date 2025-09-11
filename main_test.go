package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// newTestServer creates a router with handlers and resets the global store for isolation
func newTestServer() *mux.Router {
	store = NewInMemoryStore()
	r := mux.NewRouter()
	r.HandleFunc("/favourites/{userId}", getFavouritesHandler).Methods("GET")
	r.HandleFunc("/favourites/{userId}", addFavouriteHandler).Methods("POST")
	r.HandleFunc("/favourites/{userId}/{id}", updateFavouriteHandler).Methods("PUT", "PATCH")
	r.HandleFunc("/favourites/{userId}/{id}", deleteFavouriteHandler).Methods("DELETE")
	return r
}

func doRequest(t *testing.T, r http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != nil {
		switch b := body.(type) {
		case []byte:
			reader = bytes.NewReader(b)
		case string:
			reader = bytes.NewBufferString(b)
		default:
			data, err := json.Marshal(b)
			if err != nil {
				t.Fatalf("failed to marshal body: %v", err)
			}
			reader = bytes.NewReader(data)
		}
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func TestGetFavourites_Empty(t *testing.T) {
	r := newTestServer()
	rr := doRequest(t, r, http.MethodGet, "/favourites/user-1", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var favs []Favourite
	if err := json.Unmarshal(rr.Body.Bytes(), &favs); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if len(favs) != 0 {
		t.Fatalf("expected empty list, got %d", len(favs))
	}
}

func TestPostFavourite_CreateAndList(t *testing.T) {
	r := newTestServer()
	// Create
	payload := map[string]any{
		"assetId":     "chart-42",
		"assetType":   "chart",
		"description": "Top sales",
		"metadata":    map[string]any{"title": "Sales Q4"},
	}
	rr := doRequest(t, r, http.MethodPost, "/favourites/user-1", payload)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var created Favourite
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if created.ID == "" || created.UserID != "user-1" || created.AssetID != "chart-42" {
		t.Fatalf("unexpected created favourite: %+v", created)
	}
	// List
	rr = doRequest(t, r, http.MethodGet, "/favourites/user-1", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var favs []Favourite
	if err := json.Unmarshal(rr.Body.Bytes(), &favs); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if len(favs) != 1 || favs[0].ID != created.ID {
		t.Fatalf("expected one favourite matching created, got: %+v", favs)
	}
}

func TestPutFavourite_UpdatePreservesImmutable(t *testing.T) {
	r := newTestServer()
	// Create first
	created := createFavourite(t, r, "user-1", map[string]any{
		"assetId":     "chart-42",
		"assetType":   "chart",
		"description": "Top sales",
		"metadata":    map[string]any{"title": "Sales Q4"},
	})

	// Attempt update changing id/user/createdAt (should be ignored/preserved)
	upd := Favourite{
		ID:          "should-not-change",
		UserID:      "other-user",
		AssetID:     "chart-99",
		AssetType:   "chart",
		Description: "Updated",
		Metadata:    map[string]any{"title": "New"},
		CreatedAt:   created.CreatedAt.Add(24 * 60 * 60 * 1e9),
	}
	path := "/favourites/" + created.UserID + "/" + created.ID
	rr := doRequest(t, r, http.MethodPut, path, upd)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var got Favourite
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if got.ID != created.ID || got.UserID != created.UserID || !got.CreatedAt.Equal(created.CreatedAt) {
		t.Fatalf("immutable fields changed: before=%+v after=%+v", created, got)
	}
	if got.AssetID != "chart-99" || got.Description != "Updated" {
		t.Fatalf("mutable fields not updated: %+v", got)
	}
}

func TestPutFavourite_NotFound(t *testing.T) {
	r := newTestServer()
	upd := map[string]any{"assetId": "chart-1"}
	rr := doRequest(t, r, http.MethodPut, "/favourites/user-1/fav-999", upd)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestDeleteFavourite(t *testing.T) {
	r := newTestServer()
	created := createFavourite(t, r, "user-1", map[string]any{"assetId": "a1"})
	// Delete existing
	rr := doRequest(t, r, http.MethodDelete, "/favourites/"+created.UserID+"/"+created.ID, nil)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
	// Delete again -> 404
	rr = doRequest(t, r, http.MethodDelete, "/favourites/"+created.UserID+"/"+created.ID, nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestPostFavourite_InvalidBody(t *testing.T) {
	r := newTestServer()
	rr := doRequest(t, r, http.MethodPost, "/favourites/user-1", "not-json")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

// helper to create a favourite and parse response
func createFavourite(t *testing.T, r http.Handler, userID string, body map[string]any) Favourite {
	t.Helper()
	rr := doRequest(t, r, http.MethodPost, "/favourites/"+userID, body)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var created Favourite
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	return created
}

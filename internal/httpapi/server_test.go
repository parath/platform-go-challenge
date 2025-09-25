package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/parath/platform-go-challenge/internal/favourites"
)

// --- MockStore for testing error paths ---
type MockStore struct {
	Err error
}

func (m *MockStore) AddFavourite(favourites.Favourite) (favourites.Favourite, error) {
	return favourites.Favourite{}, m.Err
}
func (m *MockStore) GetFavourites(userID string) ([]favourites.Favourite, error) { return nil, m.Err }
func (m *MockStore) UpdateFavourite(userID, favouriteID string, update favourites.Favourite) (favourites.Favourite, error) {
	return favourites.Favourite{}, m.Err
}
func (m *MockStore) DeleteFavourite(userID, favouriteID string) error { return m.Err }
func (m *MockStore) NextFavouriteID() string                          { return "fav-mock" }

// --- Helpers for integration tests ---
func newTestServer() *mux.Router { return NewServer(favourites.NewInMemoryStore()) }

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

func createFavourite(t *testing.T, r http.Handler, userID string, body map[string]any) favourites.Favourite {
	t.Helper()
	rr := doRequest(t, r, http.MethodPost, "/favourites/"+userID, body)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var created favourites.Favourite
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	return created
}

// --- Tests ---
func TestGetFavourites_Empty(t *testing.T) {
	r := newTestServer()
	rr := doRequest(t, r, http.MethodGet, "/favourites/user-1", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var favs []favourites.Favourite
	if err := json.Unmarshal(rr.Body.Bytes(), &favs); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if len(favs) != 0 {
		t.Fatalf("expected empty list, got %d", len(favs))
	}
}

func TestPostFavourite_CreateAndList(t *testing.T) {
	r := newTestServer()
	payload := map[string]any{
		"assetId":     "chart-42",
		"assetType":   "chart",
		"description": "Top sales",
		"assetData":   map[string]any{"title": "Sales Q4"},
	}
	rr := doRequest(t, r, http.MethodPost, "/favourites/user-1", payload)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var created favourites.Favourite
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if created.ID == "" || created.UserID != "user-1" {
		t.Fatalf("unexpected created favourite: %+v", created)
	}
	// List
	rr = doRequest(t, r, http.MethodGet, "/favourites/user-1", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestPutFavourite_Update(t *testing.T) {
	r := newTestServer()
	created := createFavourite(t, r, "user-1", map[string]any{
		"assetId":     "chart-42",
		"assetType":   "chart",
		"description": "Top sales",
	})

	upd := favourites.Favourite{
		AssetID:     "chart-99",
		AssetType:   "chart",
		Description: "Updated",
		CreatedAt:   time.Now(),
	}
	path := "/favourites/" + created.UserID + "/" + created.ID
	rr := doRequest(t, r, http.MethodPut, path, upd)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestPutFavourite_NotFound(t *testing.T) {
	r := newTestServer()
	upd := map[string]any{"assetId": "chart-1", "assetType": "chart"}
	rr := doRequest(t, r, http.MethodPut, "/favourites/user-1/fav-999", upd)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}
	var errBody map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &errBody); err != nil {
		t.Fatalf("invalid json error body: %v", err)
	}
	if _, ok := errBody["error"]; !ok {
		t.Fatalf("expected error field in body, got %v", errBody)
	}
}

func TestDeleteFavourite(t *testing.T) {
	r := newTestServer()
	created := createFavourite(t, r, "user-1", map[string]any{"assetId": "a1", "assetType": "chart"})
	rr := doRequest(t, r, http.MethodDelete, "/favourites/"+created.UserID+"/"+created.ID, nil)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
	// Delete again -> 404
	rr = doRequest(t, r, http.MethodDelete, "/favourites/"+created.UserID+"/"+created.ID, nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}
}

func TestPostFavourite_InvalidBody(t *testing.T) {
	r := newTestServer()
	rr := doRequest(t, r, http.MethodPost, "/favourites/user-1", "not-json")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}
	var errBody map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &errBody); err != nil {
		t.Fatalf("invalid json error body: %v", err)
	}
}

// --- Error path tests with MockStore ---
func TestGetFavourites_StoreError(t *testing.T) {
	badStore := &MockStore{Err: errors.New("boom")}
	r := NewServer(badStore)

	rr := doRequest(t, r, http.MethodGet, "/favourites/user-1", nil)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}
}

func TestAddFavourite_StoreError(t *testing.T) {
	badStore := &MockStore{Err: errors.New("boom")}
	r := NewServer(badStore)
	payload := map[string]any{"assetId": "a1", "assetType": "chart"}
	rr := doRequest(t, r, http.MethodPost, "/favourites/user-1", payload)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}
}

// --- Conflict tests ---
func TestPostFavourite_DuplicateConflict(t *testing.T) {
	r := newTestServer()
	payload := map[string]any{"assetId": "a1", "assetType": "chart"}
	_ = doRequest(t, r, http.MethodPost, "/favourites/user-1", payload)
	rr := doRequest(t, r, http.MethodPost, "/favourites/user-1", payload)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rr.Code)
	}
}

// --- Validation tests ---
func TestPostFavourite_Validation_MissingAssetId(t *testing.T) {
	r := newTestServer()
	payload := map[string]any{
		"assetType": "chart",
		"assetData": map[string]any{"title": "X"},
	}
	rr := doRequest(t, r, http.MethodPost, "/favourites/user-1", payload)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestPostFavourite_Validation_InvalidAssetType(t *testing.T) {
	r := newTestServer()
	payload := map[string]any{
		"assetId":   "chart-1",
		"assetType": "wrong",
	}
	rr := doRequest(t, r, http.MethodPost, "/favourites/user-1", payload)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestPutFavourite_ChangeToExistingAsset_Conflict(t *testing.T) {
	r := newTestServer()
	fav1 := createFavourite(t, r, "user-1", map[string]any{"assetId": "a1", "assetType": "chart"})
	_ = createFavourite(t, r, "user-1", map[string]any{"assetId": "a2", "assetType": "chart"})
	upd := favourites.Favourite{AssetID: "a2", AssetType: "chart"}
	path := "/favourites/" + fav1.UserID + "/" + fav1.ID
	rr := doRequest(t, r, http.MethodPut, path, upd)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rr.Code)
	}
}

func TestPostFavourite_Validation_LongDescription(t *testing.T) {
	r := newTestServer()
	payload := map[string]any{
		"assetId":     "chart-1",
		"assetType":   "chart",
		"description": string(make([]byte, 600)), // >512 chars
	}
	rr := doRequest(t, r, http.MethodPost, "/favourites/user-1", payload)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestPostFavourite_Validation_AssetDataTooLarge(t *testing.T) {
	r := newTestServer()
	huge := make([]byte, 200*1024) // 200KB > 100KB limit
	payload := map[string]any{
		"assetId":   "chart-2",
		"assetType": "chart",
		"assetData": huge,
	}
	rr := doRequest(t, r, http.MethodPost, "/favourites/user-1", payload)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

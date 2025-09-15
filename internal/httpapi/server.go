package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/parath/platform-go-challenge/internal/favourites"
)

type Server struct {
	store favourites.Store
}

type ErrorCode string

const (
	CodeInvalidBody ErrorCode = "invalid_body"
	CodeStoreError  ErrorCode = "store_error"
	CodeNotFound    ErrorCode = "not_found"
	CodeConflict    ErrorCode = "conflict"
)

type apiError struct {
	Error string    `json:"error"`
	Code  ErrorCode `json:"code,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// logRequest writes a simple, consistent log line for responses (success or error).
// extras is an optional string with additional context like "status=... code=... msg=... userId=... favId=...".
func logRequest(r *http.Request, extras string) {
	if extras != "" {
		log.Printf("request: %s %s %s", r.Method, r.URL.Path, extras)
		return
	}
	log.Printf("request: %s %s", r.Method, r.URL.Path)
}

func writeError(w http.ResponseWriter, r *http.Request, status int, msg string, code ErrorCode) {
	logRequest(r, fmt.Sprintf("status=%d code=%s msg=%s", status, code, msg))
	writeJSON(w, status, apiError{Error: msg, Code: code})
}

func writeOK(w http.ResponseWriter, r *http.Request, status int, v any, extras string) {
	logRequest(r, fmt.Sprintf("status=%d ok %s", status, extras))
	writeJSON(w, status, v)
}

// validation helpers
const (
	assetTypeChart    = "chart"
	assetTypeInsight  = "insight"
	assetTypeAudience = "audience"
)

func isValidAssetType(t string) bool {
	switch t {
	case assetTypeChart, assetTypeInsight, assetTypeAudience:
		return true
	default:
		return false
	}
}

// This service stores a lightweight snapshot of the upstream asset, so we cap assetData
// around 100KB to prevent abuse.
func validateFavouriteInput(f favourites.Favourite) (string, ErrorCode) {
	if f.AssetID == "" {
		return "assetId is required", CodeInvalidBody
	}
	if !isValidAssetType(string(f.AssetType)) {
		return "invalid assetType", CodeInvalidBody
	}
	if len(f.AssetData) > 100*1024 { // ~100KB safety cap
		return "assetData too large", CodeInvalidBody
	}
	if len(f.Description) > 512 {
		return "description too long", CodeInvalidBody
	}
	return "", ""
}

func NewServer(store favourites.Store) *mux.Router {
	s := &Server{store: store}
	r := mux.NewRouter()
	r.HandleFunc("/favourites/{userId}", s.getFavouritesHandler).Methods("GET")
	r.HandleFunc("/favourites/{userId}", s.addFavouriteHandler).Methods("POST")
	r.HandleFunc("/favourites/{userId}/{id}", s.updateFavouriteHandler).Methods("PUT")
	r.HandleFunc("/favourites/{userId}/{id}", s.deleteFavouriteHandler).Methods("DELETE")
	return r
}

// Handlers
func (s *Server) getFavouritesHandler(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userId"]
	favs, err := s.store.GetFavourites(userID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "failed to get favourites", CodeStoreError)
		return
	}

	writeOK(w, r, http.StatusOK, favs, fmt.Sprintf("userId=%s", userID))
}

func (s *Server) addFavouriteHandler(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userId"]
	var f favourites.Favourite
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid request body", CodeInvalidBody)
		return
	}
	if msg, code := validateFavouriteInput(f); msg != "" {
		writeError(w, r, http.StatusBadRequest, msg, code)
		return
	}
	f.UserID = userID
	f.ID = s.store.NextFavouriteID()
	f.CreatedAt = time.Now()
	f.UpdatedAt = f.CreatedAt
	err := s.store.AddFavourite(f)
	if err != nil {
		if errors.Is(err, favourites.ErrConflict) {
			writeError(w, r, http.StatusConflict, "favourite already exists for user and asset", "conflict")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "failed to add favourite", "store_error")
		return
	}

	writeOK(w, r, http.StatusCreated, f, fmt.Sprintf("userId=%s favId=%s", f.UserID, f.ID))
}

func (s *Server) updateFavouriteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	favID := vars["id"]
	var upd favourites.Favourite
	if err := json.NewDecoder(r.Body).Decode(&upd); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid request body", CodeInvalidBody)
		return
	}
	if msg, code := validateFavouriteInput(upd); msg != "" {
		writeError(w, r, http.StatusBadRequest, msg, code)
		return
	}
	upd.UpdatedAt = time.Now()
	updated, err := s.store.UpdateFavourite(userID, favID, upd)
	if err != nil {
		if errors.Is(err, favourites.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, err.Error(), CodeNotFound)
			return
		}
		if errors.Is(err, favourites.ErrConflict) {
			writeError(w, r, http.StatusConflict, "favourite already exists for user and asset", CodeConflict)
			return
		}
		writeError(w, r, http.StatusInternalServerError, "internal server error", CodeStoreError)
		return
	}

	writeOK(w, r, http.StatusOK, updated, fmt.Sprintf("userId=%s favId=%s", userID, favID))
}

func (s *Server) deleteFavouriteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	favID := vars["id"]
	err := s.store.DeleteFavourite(userID, favID)
	if err != nil {
		if errors.Is(err, favourites.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, err.Error(), CodeNotFound)
		} else {
			writeError(w, r, http.StatusInternalServerError, "internal server error", CodeStoreError)
		}
		return
	}
	logRequest(r, fmt.Sprintf("status=%d ok userId=%s favId=%s", http.StatusNoContent, userID, favID))
	w.WriteHeader(http.StatusNoContent)
}

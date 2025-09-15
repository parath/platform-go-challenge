package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/parath/platform-go-challenge/internal/favourites"
)

type Server struct {
	store favourites.Store
}

type apiError struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, apiError{Error: msg})
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
		writeError(w, http.StatusInternalServerError, "failed to get favourites")
		return
	}

	writeJSON(w, http.StatusOK, favs)
}

func (s *Server) addFavouriteHandler(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userId"]
	var f favourites.Favourite
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	f.UserID = userID
	f.ID = s.store.NextFavouriteID()
	f.CreatedAt = time.Now()
	err := s.store.AddFavourite(f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add favourite")
		return
	}

	writeJSON(w, http.StatusCreated, f)
}

func (s *Server) updateFavouriteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	favID := vars["id"]
	var upd favourites.Favourite
	if err := json.NewDecoder(r.Body).Decode(&upd); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	updated, err := s.store.UpdateFavourite(userID, favID, upd)
	if err != nil {
		if errors.Is(err, favourites.ErrNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) deleteFavouriteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	favID := vars["id"]
	err := s.store.DeleteFavourite(userID, favID)
	if err != nil {
		if errors.Is(err, favourites.ErrNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

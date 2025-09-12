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

func NewServer(store favourites.Store) *mux.Router {
	s := &Server{store: store}
	r := mux.NewRouter()
	r.HandleFunc("/favourites/{userId}", s.getFavouritesHandler).Methods("GET")
	r.HandleFunc("/favourites/{userId}", s.addFavouriteHandler).Methods("POST")
	r.HandleFunc("/favourites/{userId}/{id}", s.updateFavouriteHandler).Methods("PUT", "PATCH")
	r.HandleFunc("/favourites/{userId}/{id}", s.deleteFavouriteHandler).Methods("DELETE")
	return r
}

// Handlers
func (s *Server) getFavouritesHandler(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userId"]
	favs, err := s.store.GetFavourites(userID)
	if err != nil {
		http.Error(w, "failed to get favourites", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(favs); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (s *Server) addFavouriteHandler(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userId"]
	var f favourites.Favourite
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	f.UserID = userID
	f.ID = s.store.NextFavouriteID()
	f.CreatedAt = time.Now()
	err := s.store.AddFavourite(f)
	if err != nil {
		http.Error(w, "failed to add favourite", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(f); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (s *Server) updateFavouriteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	favID := vars["id"]
	var upd favourites.Favourite
	if err := json.NewDecoder(r.Body).Decode(&upd); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	updated, err := s.store.UpdateFavourite(userID, favID, upd)
	if err != nil {
		if errors.Is(err, favourites.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(updated); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (s *Server) deleteFavouriteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	favID := vars["id"]
	err := s.store.DeleteFavourite(userID, favID)
	if err != nil {
		if errors.Is(err, favourites.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

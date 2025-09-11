/*
Favourites is a simple web server that manages a per-user list of favourite assets.
*/
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// Favourite represents a user's favourite asset.
// Common attributes (ID, UserID, AssetID, AssetType, Description, CreatedAt)
// are top-level fields. Asset-specific data (like chart axes or audience criteria)
// is stored as free-form JSON in Metadata for flexibility.
type Favourite struct {
	ID          string
	UserID      string
	AssetID     string
	AssetType   string
	Description string
	Metadata    map[string]interface{}
	CreatedAt   time.Time
}

// InMemoryStore stores favourites in memory per user
type InMemoryStore struct {
	mu     sync.RWMutex
	store  map[string][]Favourite
	nextID int
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{store: make(map[string][]Favourite)}
}

func (s *InMemoryStore) AddFavourite(f Favourite) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[f.UserID] = append(s.store[f.UserID], f)
}

func (s *InMemoryStore) GetFavourites(userID string) []Favourite {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Favourite(nil), s.store[userID]...)
}

func (s *InMemoryStore) UpdateFavourite(userID, favouriteID string, update Favourite) (Favourite, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.store[userID]
	for i := range list {
		if list[i].ID == favouriteID {
			// Update only mutable fields on the existing entry; keep immutable ones
			existing := list[i]
			existing.AssetID = update.AssetID
			existing.AssetType = update.AssetType
			existing.Description = update.Description
			existing.Metadata = update.Metadata
			list[i] = existing
			s.store[userID] = list
			return existing, true
		}
	}
	return Favourite{}, false
}

func (s *InMemoryStore) DeleteFavourite(userID, favouriteID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.store[userID]
	for i := range list {
		if list[i].ID == favouriteID {
			list[i] = list[len(list)-1]
			list = list[:len(list)-1]
			s.store[userID] = list
			return true
		}
	}
	return false
}

func (s *InMemoryStore) NextFavouriteID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	return fmt.Sprintf("fav-%d", s.nextID)
}

var store = NewInMemoryStore()

// Handlers
func getFavouritesHandler(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userId"]
	favs := store.GetFavourites(userID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(favs)
}

func addFavouriteHandler(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userId"]
	var f Favourite
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	f.UserID = userID
	f.ID = store.NextFavouriteID()
	f.CreatedAt = time.Now()
	store.AddFavourite(f)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(f)
}

func updateFavouriteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	favID := vars["id"]
	var upd Favourite
	if err := json.NewDecoder(r.Body).Decode(&upd); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	updated, ok := store.UpdateFavourite(userID, favID, upd)
	if !ok {
		http.Error(w, "favourite not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

func deleteFavouriteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	favID := vars["id"]
	if ok := store.DeleteFavourite(userID, favID); !ok {
		http.Error(w, "favourite not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/favourites/{userId}", getFavouritesHandler).Methods("GET")
	r.HandleFunc("/favourites/{userId}", addFavouriteHandler).Methods("POST")
	r.HandleFunc("/favourites/{userId}/{id}", updateFavouriteHandler).Methods("PUT", "PATCH")
	r.HandleFunc("/favourites/{userId}/{id}", deleteFavouriteHandler).Methods("DELETE")

	log.Println("Server listening on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}

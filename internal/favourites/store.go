package favourites

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// Store defines the behaviour for managing favourites.
type Store interface {
	AddFavourite(Favourite) error
	GetFavourites(userID string) ([]Favourite, error)
	UpdateFavourite(userID, favouriteID string, update Favourite) (Favourite, error)
	DeleteFavourite(userID, favouriteID string) error
	NextFavouriteID() string
}

// InMemoryStore stores favourites in memory per user
type InMemoryStore struct {
	mu     sync.RWMutex
	store  map[string][]Favourite
	nextID int
}

var ErrNotFound = errors.New("favourite not found")
var ErrConflict = errors.New("favourite already exists for user and asset")

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{store: make(map[string][]Favourite)}
}

func (s *InMemoryStore) AddFavourite(f Favourite) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Enforces uniqueness on (userId, assetId).
	// NOTE: With a persistent DB backend (e.g., Postgres JSONB), this uniqueness
	// would be enforced with a UNIQUE index on (userId, assetId).
	for i := range s.store[f.UserID] {
		if s.store[f.UserID][i].AssetID == f.AssetID {
			return ErrConflict
		}
	}
	f.CreatedAt = time.Now()
	f.UpdatedAt = f.CreatedAt
	s.store[f.UserID] = append(s.store[f.UserID], f)
	return nil
}

func (s *InMemoryStore) GetFavourites(userID string) ([]Favourite, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Favourite(nil), s.store[userID]...), nil
}

func (s *InMemoryStore) UpdateFavourite(userID, favouriteID string, update Favourite) (Favourite, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.store[userID]
	for i := range list {
		if list[i].ID == favouriteID {
			existing := list[i]
			// if assetId is changing, ensure no other fav uses that assetId for this user
			if existing.AssetID != update.AssetID {
				for j := range list {
					if j != i && list[j].AssetID == update.AssetID {
						return Favourite{}, ErrConflict
					}
				}
			}
			existing.AssetID = update.AssetID
			existing.AssetType = update.AssetType
			existing.Description = update.Description
			existing.AssetData = update.AssetData
			existing.UpdatedAt = time.Now()
			list[i] = existing
			s.store[userID] = list
			return existing, nil
		}
	}
	return Favourite{}, ErrNotFound
}

func (s *InMemoryStore) DeleteFavourite(userID, favouriteID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.store[userID]
	for i := range list {
		if list[i].ID == favouriteID {
			list[i] = list[len(list)-1]
			list = list[:len(list)-1]
			s.store[userID] = list
			return nil
		}
	}
	return ErrNotFound
}

func (s *InMemoryStore) NextFavouriteID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	return fmt.Sprintf("fav-%d", s.nextID)
}

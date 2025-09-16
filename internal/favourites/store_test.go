package favourites

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAddAndGetFavourites(t *testing.T) {
	store := NewInMemoryStore()

	fav1 := Favourite{
		ID:          store.NextFavouriteID(),
		UserID:      "user-1",
		AssetID:     "chart-1",
		AssetType:   "chart",
		Description: "First chart",
		CreatedAt:   time.Now(),
	}
	fav2 := Favourite{
		ID:          store.NextFavouriteID(),
		UserID:      "user-1",
		AssetID:     "chart-2",
		AssetType:   "chart",
		Description: "Second chart",
		CreatedAt:   time.Now(),
	}

	if err := store.AddFavourite(fav1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := store.AddFavourite(fav2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	favs, err := store.GetFavourites("user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(favs) != 2 {
		t.Fatalf("expected 2 favourites, got %d", len(favs))
	}
}

func TestUpdateFavourite(t *testing.T) {
	store := NewInMemoryStore()
	fav := Favourite{
		ID:          store.NextFavouriteID(),
		UserID:      "user-1",
		AssetID:     "chart-1",
		AssetType:   "chart",
		Description: "Old",
		CreatedAt:   time.Now(),
	}
	_ = store.AddFavourite(fav)

	update := Favourite{
		AssetID:     "chart-99",
		AssetType:   "chart",
		Description: "New",
		AssetData:   json.RawMessage(`{"title":"Updated"}`),
	}

	updated, err := store.UpdateFavourite("user-1", fav.ID, update)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Description != "New" || updated.AssetID != "chart-99" {
		t.Fatalf("update failed, got %+v", updated)
	}
}

func TestUpdateFavourite_NotFound(t *testing.T) {
	store := NewInMemoryStore()
	update := Favourite{AssetID: "chart-1"}
	_, err := store.UpdateFavourite("user-1", "does-not-exist", update)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteFavourite(t *testing.T) {
	store := NewInMemoryStore()
	fav := Favourite{
		ID:     store.NextFavouriteID(),
		UserID: "user-1",
	}
	_ = store.AddFavourite(fav)

	if err := store.DeleteFavourite("user-1", fav.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Deleting again should fail
	if err := store.DeleteFavourite("user-1", fav.ID); err == nil {
		t.Fatal("expected error, got nil")
	}
	if err := store.DeleteFavourite("user-1", fav.ID); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

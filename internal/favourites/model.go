package favourites

import (
	"encoding/json"
	"time"
)

type AssetType string

const (
	AssetTypeChart    AssetType = "chart"
	AssetTypeInsight  AssetType = "insight"
	AssetTypeAudience AssetType = "audience"
)

// Favourite represents a user's favourite asset.
// Common attributes (ID, UserID, AssetID, AssetType, Description, CreatedAt)
// are top-level fields. Asset-specific data (like chart axes or audience criteria)
// is stored as free-form JSON in AssetData for flexibility.
type Favourite struct {
	ID          string          `json:"id"`
	UserID      string          `json:"userId"`
	AssetID     string          `json:"assetId"`
	AssetType   AssetType       `json:"assetType"`
	Description string          `json:"description,omitempty"`
	AssetData   json.RawMessage `json:"assetData"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

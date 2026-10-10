// Package assets provides search types for TradSys v3
package assets

import (
	"strings"
	"sync"

	"github.com/abdoElHodaky/tradSys/services/exchanges"
)

// AssetSearchIndex provides indexing for asset search
type AssetSearchIndex struct {
	tokens     map[string][]string // token -> []assetIDs
	assetIndex map[string]string   // assetID -> token list
	mu         sync.RWMutex
}

// NewAssetSearchIndex creates a new asset search index
func NewAssetSearchIndex() *AssetSearchIndex {
	return &AssetSearchIndex{
		tokens:     make(map[string][]string),
		assetIndex: make(map[string]string),
	}
}

// IndexAsset adds an asset to the search index
func (idx *AssetSearchIndex) IndexAsset(assetID string, tokens []string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	for _, token := range tokens {
		idx.tokens[token] = append(idx.tokens[token], assetID)
	}
	idx.assetIndex[assetID] = strings.Join(tokens, " ")
}

// Search finds assets by token
func (idx *AssetSearchIndex) Search(token string) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return idx.tokens[token]
}

// AssetSearchQuery represents an asset search query
type AssetSearchQuery struct {
	UserID       string
	Query        string
	AssetTypes   []exchanges.AssetType
	Exchanges    []string
	Sectors      []string
	IslamicOnly  bool
	MinMarketCap float64
	MaxMarketCap float64
	Limit        int
	Offset       int
}
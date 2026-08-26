package main

import (
	"sync"
	"time"
)

var (
	catalogCache     []Product
	catalogCacheTime time.Time
	catalogCacheMu   sync.Mutex
)

const catalogTTL = 60 * time.Second

func listProductsCached() ([]Product, error) {
	catalogCacheMu.Lock()
	defer catalogCacheMu.Unlock()

	if catalogCache != nil && time.Since(catalogCacheTime) < catalogTTL {
		return catalogCache, nil
	}

	products, err := listProducts()
	if err != nil {
		return nil, err
	}

	catalogCache = products
	catalogCacheTime = time.Now()
	return products, nil
}

func invalidateCatalogCache() {
	catalogCacheMu.Lock()
	defer catalogCacheMu.Unlock()
	catalogCache = nil
}

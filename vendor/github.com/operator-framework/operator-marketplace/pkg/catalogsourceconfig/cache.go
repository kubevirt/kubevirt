package catalogsourceconfig

import (
	"sort"

	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"k8s.io/apimachinery/pkg/types"
)

// cache is an in-memory cache of CatalogSourceConfig UID : Spec.
// Note: This is a temporary construct which will be removed when we move to
// using the Operator Registry as the data store for CatalogSources. If this
// is required even after, then it should be replaced with an existing thread
// safe caching library like go-cache or cash.
//
// TODO: Make the cache app-registry version aware so that IsEntryStale() will
// fire even for the scenario where a Quay namespace has changed without
// app-registry repositories being added or removed but with existing
// repositories being updated.
type cache struct {
	entries map[types.UID]*marketplace.CatalogSourceConfigSpec
}

// Cache is the interface for the CatalogSourceConfig caching functions.
type Cache interface {
	// Get returns the cached CatalogSourceConfigSpec of the CatalogSourceConfig
	// object if it is present in the cache. The bool value indicates if the
	// Spec for the object was in the cache or not.
	Get(csc *marketplace.CatalogSourceConfig) (*marketplace.CatalogSourceConfigSpec, bool)

	// IsEntryStale figures out if the CatalogSourceConfigSpec in the
	// CatalogSourceConfig object matches its entry in the cache. Cache is
	// considered stale if it does not match. pkgStale is true then the Packages
	// have changed. If targetStale is true then the TargetNamespace has
	// changed. This implies that pkgStale is also true.
	IsEntryStale(csc *marketplace.CatalogSourceConfig) (pkgStale bool, targetStale bool)

	// Evict removes the entry for the CatalogSourceConfig object from the cache.
	Evict(csc *marketplace.CatalogSourceConfig)

	// Set adds the CatalogSourceConfigSpec for the CatalogSourceConfig object
	// into the cache.
	Set(csc *marketplace.CatalogSourceConfig)
}

func (c *cache) Get(csc *marketplace.CatalogSourceConfig) (*marketplace.CatalogSourceConfigSpec, bool) {
	entry, found := c.entries[csc.ObjectMeta.UID]
	if !found {
		return &marketplace.CatalogSourceConfigSpec{}, false
	}
	return entry, true
}

func (c *cache) IsEntryStale(csc *marketplace.CatalogSourceConfig) (bool, bool) {
	spec, found := c.Get(csc)
	// Found is false if the CSC wasn't found in the cache. So it must be stale.
	if !found {
		return true, true
	}

	if spec.TargetNamespace != csc.Spec.TargetNamespace {
		return true, true
	}

	cachedPackages := spec.GetPackageIDs()
	inPackageIDs := csc.GetPackageIDs()

	if len(cachedPackages) != len(inPackageIDs) {
		return true, false
	}

	sort.Strings(cachedPackages)
	sort.Strings(inPackageIDs)
	for i, v := range cachedPackages {
		if v != inPackageIDs[i] {
			return true, false
		}
	}
	return false, false
}

func (c *cache) Evict(csc *marketplace.CatalogSourceConfig) {
	UID := csc.ObjectMeta.UID
	_, found := c.entries[UID]
	if !found {
		return
	}
	delete(c.entries, UID)
}

func (c *cache) Set(csc *marketplace.CatalogSourceConfig) {
	c.entries[csc.ObjectMeta.UID] = &marketplace.CatalogSourceConfigSpec{
		Packages:        csc.GetPackages(),
		TargetNamespace: csc.Spec.TargetNamespace,
	}
}

// NewCache returns an initialized Cache
func NewCache() Cache {
	return &cache{
		entries: make(map[types.UID]*marketplace.CatalogSourceConfigSpec),
	}
}

package testutils

import (
	"k8s.io/client-go/tools/cache"
)

// FakeStore wraps cache.FakeCustomStore to satisfy the cache.Store interface,
// which gained Bookmark and LastStoreSyncResourceVersion methods that
// FakeCustomStore does not implement.
type FakeStore struct {
	cache.FakeCustomStore
}

func (f *FakeStore) Bookmark(_ string) {}

func (f *FakeStore) LastStoreSyncResourceVersion() string { return "" }

package testutils

import (
	"k8s.io/client-go/tools/cache"
)

// FakeStore wraps cache.FakeCustomStore to satisfy the cache.Store
// interface, which gained the Bookmark and LastStoreSyncResourceVersion
// methods in client-go v0.36 (kubernetes/kubernetes#134827) without
// FakeCustomStore being updated.
//
// TODO: drop this once cache.FakeCustomStore implements cache.Store
// upstream.
type FakeStore struct {
	cache.FakeCustomStore
	BookmarkFunc                     func(rv string)
	LastStoreSyncResourceVersionFunc func() string
}

func (f *FakeStore) Bookmark(rv string) {
	if f.BookmarkFunc != nil {
		f.BookmarkFunc(rv)
	}
}

func (f *FakeStore) LastStoreSyncResourceVersion() string {
	if f.LastStoreSyncResourceVersionFunc != nil {
		return f.LastStoreSyncResourceVersionFunc()
	}
	return ""
}

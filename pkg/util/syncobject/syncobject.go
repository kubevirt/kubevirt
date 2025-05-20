package syncobject

import "sync"

type SyncObject[T any] interface {
	Get() T
	Set(value T)
}

func NewSyncObject[T any]() SyncObject[T] {
	return &defaultSyncObject[T]{}
}

type defaultSyncObject[T any] struct {
	value T
	mu    sync.RWMutex
}

func (d *defaultSyncObject[T]) Get() T {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.value
}

func (d *defaultSyncObject[T]) Set(value T) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.value = value
}

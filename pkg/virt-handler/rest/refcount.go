package rest

import "sync"

type RefCounter[K comparable, T any] interface {
	Get(key K, createFn func() (T, func(), error)) (T, func(), error)
}

type refCounter[K comparable, T any] struct {
	lock       sync.Mutex
	references map[K]*refObj[T]
}

type refObj[T any] struct {
	count     int
	obj       T
	destroyFn func()
}

func NewRefCounter[K comparable, T any]() RefCounter[K, T] {
	return &refCounter[K, T]{
		references: make(map[K]*refObj[T]),
	}
}

func (r *refCounter[K, T]) Get(key K, createFn func() (T, func(), error)) (T, func(), error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if ref, ok := r.references[key]; ok {
		ref.count++
		return ref.obj, func() { r.releaseFunc(key) }, nil
	}

	obj, destroyFn, err := createFn()
	if err != nil {
		var zero T
		return zero, nil, err
	}

	r.references[key] = &refObj[T]{
		count:     1,
		obj:       obj,
		destroyFn: destroyFn,
	}

	return obj, func() { r.releaseFunc(key) }, nil
}

func (r *refCounter[K, T]) releaseFunc(key K) {
	r.lock.Lock()
	defer r.lock.Unlock()

	ref, ok := r.references[key]
	if !ok {
		panic("Tried to release non-existing object")
	}

	ref.count--
	if ref.count <= 0 {
		delete(r.references, key)
		if ref.destroyFn != nil {
			ref.destroyFn()
		}
	}
}

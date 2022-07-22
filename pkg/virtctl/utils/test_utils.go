package utils

import "sync"

type AtomicBool struct {
	Lock  *sync.Mutex
	value bool
}

func (b *AtomicBool) IsTrue() bool {
	b.Lock.Lock()
	defer b.Lock.Unlock()
	return b.value
}

func (b *AtomicBool) True() {
	b.Lock.Lock()
	defer b.Lock.Unlock()
	b.value = true
}

func (b *AtomicBool) False() {
	b.Lock.Lock()
	defer b.Lock.Unlock()
	b.value = false
}

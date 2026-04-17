package migrationproxy

import (
	"fmt"
	"sync"
)

type PortRange struct {
	First int
	Last  int

	used map[int]struct{}
	mu   *sync.Mutex
}

func NewPortRange(first, last int) (*PortRange, error) {
	if first < 0 || last < 0 || first > 65535 || last > 65535 || first > last {
		return nil, fmt.Errorf("invalid port range: %d-%d", first, last)
	}
	return &PortRange{
		First: first,
		Last:  last,
		used:  make(map[int]struct{}),
		mu:    &sync.Mutex{},
	}, nil
}

func (r *PortRange) AvailableCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	total := r.Last - r.First + 1
	if total <= 0 {
		return 0
	}
	used := len(r.used)
	if used >= total {
		return 0
	}
	return total - used
}

func (r *PortRange) Alloc() (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for p := r.First; p <= r.Last; p++ {
		if _, ok := r.used[p]; ok {
			continue
		}
		r.used[p] = struct{}{}
		return p, nil
	}

	return -1, fmt.Errorf("failed to allocate port in range: %d-%d", r.First, r.Last)
}

func (r *PortRange) Free(port int) {
	if port < r.First || port > r.Last {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.used[port]; ok {
		delete(r.used, port)
	}
}

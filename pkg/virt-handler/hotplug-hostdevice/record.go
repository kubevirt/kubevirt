package hotplug_hostdevice

import (
	"fmt"
	"os"
	"sync"

	"k8s.io/apimachinery/pkg/types"

	virtcache "kubevirt.io/kubevirt/pkg/virt-handler/cache"
)

type recordManager struct {
	checkpointManager virtcache.IterableCheckpointManager
	records           map[types.UID]record
	mu                sync.RWMutex
}

func newRecordManager(base string) *recordManager {
	checkpointManager := virtcache.NewIterableCheckpointManager(base)
	records := make(map[types.UID]record)
	for _, key := range checkpointManager.ListKeys() {
		record := record{}
		if err := checkpointManager.Get(key, &record); err != nil {
			continue
		}
		records[types.UID(key)] = record
	}
	return &recordManager{
		checkpointManager: checkpointManager,
		records:           records,
	}
}

func (m *recordManager) Get(vmiUID types.UID) (record, bool, error) {
	m.mu.RLock()
	r, ok := m.records[vmiUID]
	m.mu.RUnlock()
	if !ok {
		record := record{}
		if err := m.checkpointManager.Get(string(vmiUID), &record); err != nil {
			if os.IsNotExist(err) {
				return record, false, nil
			}
			return record, false, err
		}

		m.mu.Lock()
		defer m.mu.Unlock()

		m.records[vmiUID] = record
		return record, true, nil
	}
	return r, ok, nil
}

func (m *recordManager) Store(vmiUID types.UID, record record) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.checkpointManager.Store(string(vmiUID), record); err != nil {
		return fmt.Errorf("failed to store checkpoint %s, %w", vmiUID, err)
	}
	m.records[vmiUID] = record
	return nil
}

func (m *recordManager) StoreEntry(vmiUID types.UID, targetFile string) error {
	m.mu.RLock()
	r, ok := m.records[vmiUID]
	m.mu.RUnlock()

	if !ok {
		r = record{}
	}

	entry := recordEntry{
		TargetFile: targetFile,
	}

	if r.Has(entry) {
		return nil
	}

	r.Add(entry)
	return m.Store(vmiUID, r)
}

func (m *recordManager) Delete(vmiUID types.UID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.checkpointManager.Delete(string(vmiUID)); err != nil {
		return fmt.Errorf("failed to delete checkpoint %s, %w", vmiUID, err)
	}
	delete(m.records, vmiUID)
	return nil
}

type record struct {
	Entries []recordEntry `json:"entries"`
}

func (r *record) Has(entry recordEntry) bool {
	for _, e := range r.Entries {
		if e.Equals(entry) {
			return true
		}
	}
	return false
}

func (r *record) Add(entry recordEntry) {
	for i, e := range r.Entries {
		if e.Equals(entry) {
			r.Entries[i] = entry
			return
		}
	}
	r.Entries = append(r.Entries, entry)
}

type recordEntry struct {
	TargetFile string `json:"targetFile"`
}

func (r recordEntry) Equals(other recordEntry) bool {
	return r.TargetFile == other.TargetFile
}

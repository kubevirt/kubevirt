package status

import (
	"errors"
	"sync"

	log "github.com/sirupsen/logrus"
)

// SyncRatio provides an interface for managing syncs which are used to report
// operator status to CVO.
type SyncRatio interface {
	GetSyncs() (failedSyncs int, syncEvents int)
	ReportFailedSync()
	ReportSyncEvent()
	IsSucceeding() (bool, *float32)
}

// NewSyncRatio returns a syncRatio object or an error if invalid parameters
// are provided.
func NewSyncRatio(successRatio float32, syncsBeforeTruncate int, syncTruncateValue int) (SyncRatio, error) {
	errMsg := ""
	if successRatio < 0 || successRatio > 1 {
		errMsg += "successRatio must be greater than or equal to 0 and less than or equal to 1. "
	}
	if syncsBeforeTruncate <= 0 {
		errMsg += "syncsBeforeTruncate must be greater than 0. "
	}
	if syncTruncateValue <= 0 {
		errMsg += "syncTruncateValue must be greater than 0."
	}
	if errMsg != "" {
		return nil, errors.New(errMsg)
	}

	return &syncRatio{
		successRatio:        successRatio,
		syncsBeforeTruncate: syncsBeforeTruncate,
		syncTruncateValue:   syncTruncateValue,
	}, nil
}

type syncRatio struct {
	// successRatio is the ratio of successfulSyncs to syncEvents
	successRatio float32

	// syncsBeforeTruncate the number of syncEvents before failedSyncs and
	// syncEvents are truncated.
	syncsBeforeTruncate int

	// The value that failedSyncs and syncEvents will be truncated by.
	syncTruncateValue int

	// failedSyncs represents the number of failed syncs.
	failedSyncs int
	// syncEvents represents the sum of syncs events.
	syncEvents int

	// lock is used to prevent race condition on syncEvents and failedSyncs fields.
	lock sync.Mutex
}

// GetSyncs returns the number of failedSyncs and the number of syncEvents
func (s *syncRatio) GetSyncs() (failedSyncs int, syncEvents int) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.failedSyncs, s.syncEvents
}

// ReportFailedSync increments the number of syncEvents by one
func (s *syncRatio) ReportFailedSync() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.failedSyncs++
}

// ReportSyncEvent increments the number of syncEvents
func (s *syncRatio) ReportSyncEvent() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.syncEvents++
}

// truncateSyncs is used to prevent failedSyncs and syncEvents from overflowing.
func (s *syncRatio) truncateSyncs() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.syncEvents = s.syncEvents % s.syncEvents
	s.failedSyncs = s.failedSyncs % s.syncTruncateValue
}

// isSucceeding returns whether or not the cluster operator should report a successful state.
// If syncEvents is less than or equal to 0 an error is returned and syncr.
func (s *syncRatio) IsSucceeding() (bool, *float32) {
	ratio := s.getRatio()
	// return error if s.syncs <= 0
	if ratio == nil {
		return false, ratio
	}

	if *ratio >= s.successRatio {
		return true, ratio
	}
	return false, ratio
}

// getRatio returns the ratio of successfulSyncs to syncEvents
// If s.syncs is equal to 0 then nil is returned
func (s *syncRatio) getRatio() *float32 {
	// Prevent number of syncs from growing indefinitely
	if s.syncEvents > s.syncsBeforeTruncate {
		s.truncateSyncs()
	}
	if s.syncEvents <= 0 {
		return nil
	}
	ratio := float32(s.syncEvents-s.failedSyncs) / float32(s.syncEvents)
	log.Debugf("[status] Successful syncs to total syncs ratio: %v", ratio)
	return &ratio
}

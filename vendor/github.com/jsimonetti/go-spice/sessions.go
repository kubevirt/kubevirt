package spice

import (
	"fmt"
	"sync"
)

// sessionTable holds a mapping of SessionID and destination node computeAddress
// map[sessionid]computeAddress
type sessionTable struct {
	lock    sync.Mutex
	entries map[uint32]*sessionEntry
}

type sessionEntry struct {
	computeAddress string
	usageCount     int
	authToken      string
}

func newSessionTable() *sessionTable {
	return &sessionTable{
		entries: make(map[uint32]*sessionEntry),
	}
}

func (s *sessionTable) Lookup(session uint32) bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	_, ok := s.entries[session]
	return ok
}

func (s *sessionTable) OTP(session uint32) string {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, ok := s.entries[session]; ok {
		return s.entries[session].authToken
	}
	return ""
}

func (s *sessionTable) Compute(session uint32) string {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, ok := s.entries[session]; ok {
		return s.entries[session].computeAddress
	}
	return ""
}

func (s *sessionTable) Add(session uint32, destination string, otp string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, ok := s.entries[session]; !ok {
		s.entries[session] = &sessionEntry{computeAddress: destination, usageCount: 1, authToken: otp}
	}
	return
}

func (s *sessionTable) Connect(session uint32) (string, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, ok := s.entries[session]; !ok {
		return "", fmt.Errorf("no such session in table")
	}
	s.entries[session].usageCount = s.entries[session].usageCount + 1
	return s.entries[session].computeAddress, nil
}

func (s *sessionTable) Disconnect(session uint32) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, ok := s.entries[session]; !ok {
		return
	}
	s.entries[session].usageCount = s.entries[session].usageCount - 1
	if s.entries[session].usageCount < 1 {
		delete(s.entries, session)
	}
	return
}

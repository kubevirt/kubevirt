package filewatcher

import (
	"errors"
	"sync"
	"syscall"
	"time"
)

type Event uint32

const (
	Create Event = 1 << iota
	Remove
	InoChange
)

type FileWatcher struct {
	path     string
	interval time.Duration

	Events chan Event
	Errors chan error
	done   chan struct{}

	lastIno uint64
	closeMu sync.Mutex
}

func New(path string, interval time.Duration) *FileWatcher {
	return &FileWatcher{
		path:     path,
		interval: interval,
		Events:   make(chan Event),
		Errors:   make(chan error),
		done:     make(chan struct{}),
	}
}

func (f *FileWatcher) Run() {
	f.statFirst()
	go func() {
		defer close(f.Events)
		defer close(f.Errors)

		ticker := time.Tick(f.interval)
		for {
			select {
			case <-f.done:
				return
			case <-ticker:
				f.stat()
			}
		}
	}()
}

func (f *FileWatcher) Close() {
	f.closeMu.Lock()
	if f.IsClosed() {
		f.closeMu.Unlock()
		return
	}
	close(f.done)
	f.closeMu.Unlock()
	return
}

func (f *FileWatcher) statFirst() {
	stat := &syscall.Stat_t{}
	if err := syscall.Stat(f.path, stat); err != nil {
		if !errors.Is(err, syscall.ENOENT) {
			f.sendError(err)
		}
		return
	}
	f.lastIno = stat.Ino
}

func (f *FileWatcher) stat() {
	stat := &syscall.Stat_t{}
	if err := syscall.Stat(f.path, stat); err != nil {
		if errors.Is(err, syscall.ENOENT) {
			if f.lastIno != 0 {
				f.lastIno = 0
				f.sendEvent(Remove)
			}
		} else {
			f.sendError(err)
		}
		return
	}

	if f.lastIno == 0 {
		f.lastIno = stat.Ino
		f.sendEvent(Create)
		return
	}

	if stat.Ino != f.lastIno {
		f.lastIno = stat.Ino
		f.sendEvent(InoChange)
	}
}

func (f *FileWatcher) sendEvent(e Event) bool {
	select {
	case f.Events <- e:
		return true
	case <-f.done:
		return false
	}
}

func (f *FileWatcher) sendError(err error) bool {
	select {
	case f.Errors <- err:
		return true
	case <-f.done:
		return false
	}
}

func (f *FileWatcher) IsClosed() bool {
	select {
	case <-f.done:
		return true
	default:
		return false
	}
}

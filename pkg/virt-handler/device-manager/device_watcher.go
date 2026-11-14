/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package device_manager

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"unsafe"

	"github.com/fsnotify/fsnotify"
	"golang.org/x/sys/unix"
)

// deviceEventMask is the inotify mask used to watch the directories holding
// host devices. It deliberately excludes IN_MODIFY and IN_ATTRIB: the device
// plugin health checks only ever react to devices appearing, disappearing or
// being renamed, so subscribing to write/attribute events would make every
// write(2) to a block device in the watched directory queue an event and wake
// virt-handler for nothing.
//
// fsnotify's Watcher.Add always uses a mask that includes IN_MODIFY (0xfc6 in
// fsnotify v1.9.0), and that mask cannot be narrowed through the public API
// because the withOps option is not exported (see fsnotify issue #764 and the
// still-open export PR fsnotify/fsnotify#776). Until that lands, use inotify
// directly with the narrow mask below and emit the same fsnotify.Event values
// the previous implementation produced.
const deviceEventMask = unix.IN_CREATE |
	unix.IN_DELETE |
	unix.IN_DELETE_SELF |
	unix.IN_MOVED_FROM |
	unix.IN_MOVED_TO |
	unix.IN_MOVE_SELF

const (
	// inotifyEventSize and inotifyEventHeaderSize describe the fixed size of
	// the inotify_event structure (see inotify(7)).
	inotifyEventSize = unix.SizeofInotifyEvent
)

// deviceWatcher is a narrow-mask drop-in replacement for the subset of
// fsnotify.Watcher the device plugin health checks use. It exposes the same
// Events, Errors and Close members plus an Add method so the call sites stay
// unchanged.
type deviceWatcher struct {
	fd int

	Events chan fsnotify.Event
	Errors chan error

	closeOnce sync.Once
	done      chan struct{}

	// wdToPath maps an inotify watch descriptor back to the path it was
	// registered for, so events can report a full path the same way fsnotify
	// does.
	mutex    sync.RWMutex
	wdToPath map[int]string
}

// newDeviceWatcher creates a watcher backed by inotify with deviceEventMask.
func newDeviceWatcher() (*deviceWatcher, error) {
	// IN_CLOEXEC so the fd is not leaked into children, IN_NONBLOCK so the
	// reader goroutine can be released by closing the fd.
	fd, err := unix.InotifyInit1(unix.IN_CLOEXEC | unix.IN_NONBLOCK)
	if err != nil {
		return nil, fmt.Errorf("failed to create inotify instance: %v", err)
	}

	w := &deviceWatcher{
		fd:       fd,
		Events:   make(chan fsnotify.Event),
		Errors:   make(chan error),
		done:     make(chan struct{}),
		wdToPath: make(map[int]string),
	}

	go w.readEvents()

	return w, nil
}

// Add starts watching the named directory or file with the narrow mask.
func (w *deviceWatcher) Add(name string) error {
	// IN_MASK_ADD is intentionally not set so each Add replaces the mask for
	// an already watched inode; the mask is identical for every call anyway.
	wd, err := unix.InotifyAddWatch(w.fd, name, deviceEventMask)
	if err != nil {
		return err
	}

	w.mutex.Lock()
	w.wdToPath[wd] = name
	w.mutex.Unlock()

	return nil
}

// Close removes all watches and stops the reader goroutine.
func (w *deviceWatcher) Close() error {
	var err error
	w.closeOnce.Do(func() {
		close(w.done)
		err = unix.Close(w.fd)
		close(w.Events)
		close(w.Errors)
	})
	return err
}

func (w *deviceWatcher) readEvents() {
	buf := make([]byte, inotifyEventSize*4096)

	for {
		select {
		case <-w.done:
			return
		default:
		}

		n, err := unix.Read(w.fd, buf)
		if err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}
			// The fd is closed on Close(), which surfaces as EBADF; treat
			// that as a clean shutdown rather than an error.
			if errors.Is(err, unix.EBADF) {
				return
			}
			if errors.Is(err, unix.EAGAIN) {
				// Non-blocking fd with no data ready: wait briefly and retry.
				if !w.waitForData() {
					return
				}
				continue
			}
			w.sendError(fmt.Errorf("failed to read inotify events: %v", err))
			return
		}

		if n < inotifyEventSize {
			continue
		}

		for offset := 0; offset+inotifyEventSize <= n; {
			raw := (*unix.InotifyEvent)(unsafe.Pointer(&buf[offset]))
			eventLen := inotifyEventSize + int(raw.Len)

			if raw.Mask&unix.IN_Q_OVERFLOW != 0 {
				w.sendError(fmt.Errorf("inotify queue overflowed, events may have been missed"))
			} else {
				w.sendEvent(w.parseEvent(raw, buf, offset))
			}

			offset += eventLen
		}
	}
}

func (w *deviceWatcher) parseEvent(raw *unix.InotifyEvent, buf []byte, offset int) fsnotify.Event {
	var name string
	if raw.Len > 0 {
		// The name is a NUL-terminated, NUL-padded sequence of raw.Len bytes
		// directly after the fixed-size inotify_event header.
		nameBytes := buf[offset+inotifyEventSize : offset+inotifyEventSize+int(raw.Len)]
		name = strings.TrimRight(string(nameBytes), "\x00")
	}

	w.mutex.RLock()
	base, ok := w.wdToPath[int(raw.Wd)]
	w.mutex.RUnlock()

	fullPath := name
	if ok {
		if name != "" {
			fullPath = base + string(os.PathSeparator) + name
		} else {
			fullPath = base
		}
	}

	return fsnotify.Event{
		Name: fullPath,
		Op:   opFromMask(raw.Mask),
	}
}

func opFromMask(mask uint32) fsnotify.Op {
	var op fsnotify.Op
	if mask&(unix.IN_CREATE|unix.IN_MOVED_TO) != 0 {
		op |= fsnotify.Create
	}
	if mask&(unix.IN_DELETE|unix.IN_DELETE_SELF|unix.IN_IGNORED) != 0 {
		op |= fsnotify.Remove
	}
	if mask&(unix.IN_MOVED_FROM|unix.IN_MOVE_SELF) != 0 {
		op |= fsnotify.Rename
	}
	return op
}

// waitForData blocks until the inotify fd becomes readable or the watcher is
// closed. It is used because the fd is non-blocking.
func (w *deviceWatcher) waitForData() bool {
	pfd := []unix.PollFd{{Fd: int32(w.fd), Events: unix.POLLIN}}

	for {
		_, err := unix.Poll(pfd, 1000)
		if err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}
			return false
		}
		if pfd[0].Revents&unix.POLLIN != 0 {
			return true
		}
		select {
		case <-w.done:
			return false
		default:
			return true
		}
	}
}

func (w *deviceWatcher) sendEvent(event fsnotify.Event) {
	select {
	case w.Events <- event:
	case <-w.done:
	}
}

func (w *deviceWatcher) sendError(err error) {
	select {
	case w.Errors <- err:
	case <-w.done:
	}
}

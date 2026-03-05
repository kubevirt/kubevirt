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

package conntrack

import (
	"bufio"
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/client-go/log"
)

const SyncTimeout = 200 * time.Millisecond

type targetState struct {
	proxyListener net.Listener
	hookListener  net.Listener
	injectionDone chan struct{}
	cancel        context.CancelFunc
}

type TargetHandler struct {
	ciliumClient ConntrackClient
	mu           sync.Mutex
	states       map[types.UID]*targetState
}

func NewTargetHandler(ciliumClient ConntrackClient) *TargetHandler {
	return &TargetHandler{
		ciliumClient: ciliumClient,
		states:       make(map[types.UID]*targetState),
	}
}

func (h *TargetHandler) StartProxyListener(vmiUID types.UID, socketPath string) error {
	h.mu.Lock()
	state := h.getOrCreateState(vmiUID)
	if state.proxyListener != nil {
		h.mu.Unlock()
		return nil
	}
	h.mu.Unlock()

	listener, err := listenUnix(socketPath)
	if err != nil {
		return err
	}

	h.mu.Lock()
	state.proxyListener = listener
	h.mu.Unlock()

	go h.acceptConnection(vmiUID, listener, h.handleProxyConnection)

	log.Log.V(3).Infof("Conntrack sync: started proxy listener at %s for VMI %s", socketPath, vmiUID)
	return nil
}

func (h *TargetHandler) StartHookListener(vmiUID types.UID, socketPath string) error {
	h.mu.Lock()
	state := h.getOrCreateState(vmiUID)
	if state.hookListener != nil {
		h.mu.Unlock()
		return nil
	}
	h.mu.Unlock()

	listener, err := listenUnix(socketPath)
	if err != nil {
		return err
	}

	h.mu.Lock()
	state.hookListener = listener
	h.mu.Unlock()

	go h.acceptConnection(vmiUID, listener, h.handleHookConnection)

	log.Log.V(3).Infof("Conntrack sync: started hook listener at %s for VMI %s", socketPath, vmiUID)
	return nil
}

func listenUnix(socketPath string) (net.Listener, error) {
	if err := os.MkdirAll(filepath.Dir(socketPath), 0755); err != nil {
		return nil, err
	}
	os.Remove(socketPath)
	return net.Listen("unix", socketPath)
}

func (h *TargetHandler) acceptConnection(vmiUID types.UID, listener net.Listener, handler func(types.UID, net.Conn)) {
	conn, err := listener.Accept()
	if err != nil {
		if !errors.Is(err, net.ErrClosed) {
			log.Log.Warningf("Conntrack sync: accept error for VMI %s: %v", vmiUID, err)
		}
		return
	}
	handler(vmiUID, conn)
}

func (h *TargetHandler) handleProxyConnection(vmiUID types.UID, conn net.Conn) {
	defer conn.Close()

	msg, err := DecodeSyncMessage(conn)
	if err != nil {
		log.Log.Warningf("Conntrack sync: failed to decode message for VMI %s: %v", vmiUID, err)
		return
	}

	log.Log.V(3).Infof("Conntrack sync: received %d bytes for VMI %s", len(msg.Data), vmiUID)

	h.onCTReceived(vmiUID, msg)
}

func (h *TargetHandler) onCTReceived(vmiUID types.UID, msg *SyncMessage) {
	h.mu.Lock()
	state := h.getOrCreateState(vmiUID)
	ctx, cancel := context.WithCancel(context.Background())
	state.cancel = cancel
	h.mu.Unlock()

	err := h.ciliumClient.ImportConntrack(ctx, msg.Data, msg.Version)

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.states[vmiUID] == nil {
		return
	}

	if ctx.Err() == context.DeadlineExceeded {
		log.Log.Warningf("Conntrack sync: import timed out for VMI %s", vmiUID)
	} else if err != nil {
		log.Log.Warningf("Conntrack sync: import failed for VMI %s: %v", vmiUID, err)
	} else {
		log.Log.V(3).Infof("Conntrack sync: import completed for VMI %s", vmiUID)
	}

	close(state.injectionDone)
}

func (h *TargetHandler) handleHookConnection(vmiUID types.UID, conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "wait") {
			h.onHookSignal(vmiUID)
		}
	}

	conn.Write([]byte("ok\n"))
}

func (h *TargetHandler) onHookSignal(vmiUID types.UID) {
	h.mu.Lock()
	state, exists := h.states[vmiUID]
	if !exists {
		h.mu.Unlock()
		return
	}
	done := state.injectionDone
	h.mu.Unlock()

	select {
	case <-done:
	case <-time.After(SyncTimeout):
		h.mu.Lock()
		if state.cancel != nil {
			state.cancel()
		}
		h.mu.Unlock()
		log.Log.Warningf("Conntrack sync: hook timeout (%v) for VMI %s", SyncTimeout, vmiUID)
	}
}

func (h *TargetHandler) Cleanup(vmiUID types.UID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	state, exists := h.states[vmiUID]
	if !exists {
		return
	}

	if state.proxyListener != nil {
		state.proxyListener.Close()
	}
	if state.hookListener != nil {
		state.hookListener.Close()
	}
	if state.cancel != nil {
		state.cancel()
	}

	delete(h.states, vmiUID)

	log.Log.V(3).Infof("Conntrack sync: cleaned up target state for VMI %s", vmiUID)
}

func (h *TargetHandler) getOrCreateState(vmiUID types.UID) *targetState {
	state, exists := h.states[vmiUID]
	if !exists {
		state = &targetState{
			injectionDone: make(chan struct{}),
		}
		h.states[vmiUID] = state
	}
	return state
}

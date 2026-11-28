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

package dmetrics_manager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/clock"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/downwardmetrics"
	virtioserial "kubevirt.io/kubevirt/pkg/downwardmetrics/virtio-serial"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

const (
	baseRetryDelay      = 5 * time.Second  // first retry delay
	maxRetryDelay       = 5 * time.Minute  // backoff cap
	maxRetryShift       = 6                // caps the exponent to avoid overflow
	healthyRunThreshold = 30 * time.Second // serving time that resets the backoff
)

func NewDownwardMetricsManager(nodeName string) *DownwardMetricsManager {
	return &DownwardMetricsManager{
		done:       false,
		nodeName:   nodeName,
		stopServer: make(map[types.UID]context.CancelFunc),
		runServer:  virtioserial.RunDownwardMetricsVirtioServer,
		retry:      make(map[types.UID]*retryState),
		clock:      clock.RealClock{},
	}
}

// retryState is the restart backoff state for one VMI.
type retryState struct {
	failures  int       // failed restarts since the last healthy run
	nextStart time.Time // earliest allowed restart
}

// DownwardMetricsManager controls the lifetime of the DownwardMetrics servers.
// Each server is tied to the lifetime of the VMI and DownwardMetricsManager itself.
type DownwardMetricsManager struct {
	lock       sync.Mutex
	done       bool
	nodeName   string
	stopServer map[types.UID]context.CancelFunc
	runServer  func(context.Context, string, string, string, func(time.Duration))
	retry      map[types.UID]*retryState
	clock      clock.Clock
}

// Run blocks until stopCh is closed. When done, it stops all remaining
// running DownwardMetrics servers.
func (m *DownwardMetricsManager) Run(stopCh chan struct{}) {
	defer m.stop()
	<-stopCh
}

func (m *DownwardMetricsManager) stop() {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.done = true

	// Stop all DownwardMetrics servers
	for vmiUID, stopServerFn := range m.stopServer {
		stopServerFn()
		delete(m.stopServer, vmiUID)
	}
	m.retry = nil
}

// StopServer removes the VMI UID from the list of served VMs and stops its server.
func (m *DownwardMetricsManager) StopServer(vmi *v1.VirtualMachineInstance) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.done {
		return
	}

	// Even it's not strictly required for stopping the server,
	// since the server will stop when the VM closes the unix socket,
	// we cancel the context to avoid leaking it
	if cancelCtx, exists := m.stopServer[vmi.UID]; exists {
		cancelCtx()
		delete(m.stopServer, vmi.UID)
	}
	delete(m.retry, vmi.UID)
}

// StartServer starts a server if requested by a running VMI and not already started.
// It calls requestRetry after an unexpected server exit, outside the manager lock.
// Restarts honor a per-VMI backoff; an early sync reschedules the remaining delay.
func (m *DownwardMetricsManager) StartServer(vmi *v1.VirtualMachineInstance, pid int, requestRetry func(time.Duration)) error {
	if !downwardmetrics.HasDevice(&vmi.Spec) || !vmi.IsRunning() {
		return nil
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	if m.done {
		return nil
	}

	if _, alreadyStarted := m.stopServer[vmi.UID]; alreadyStarted {
		return nil
	}

	if state, ok := m.retry[vmi.UID]; ok {
		if remaining := state.nextStart.Sub(m.clock.Now()); remaining > 0 {
			requestRetry(remaining)
			return nil
		}
	}

	launcherSocketPath, err := cmdclient.FindSocket(vmi)
	if err != nil {
		return fmt.Errorf("failed to get the launcher socket for VMI [%s], error: %v", vmi.GetName(), err)
	}

	channelPath := downwardmetrics.ChannelSocketPathOnHost(pid)
	ctx, cancelCtx := context.WithCancel(context.Background())

	onExit := func(servedFor time.Duration) {
		m.lock.Lock()
		// A cancelled attempt must not remove a replacement's entry.
		if m.done || ctx.Err() != nil {
			m.lock.Unlock()
			return
		}
		cancelCtx()
		delete(m.stopServer, vmi.UID)
		delay := m.nextRetryDelay(vmi.UID, servedFor)
		m.lock.Unlock()

		requestRetry(delay)
	}
	m.runServer(ctx, m.nodeName, channelPath, launcherSocketPath, onExit)
	m.stopServer[vmi.UID] = cancelCtx

	return nil
}

// nextRetryDelay returns the next restart delay and updates the deadline. A run of at
// least healthyRunThreshold resets the backoff; otherwise it grows. Callers hold m.lock.
func (m *DownwardMetricsManager) nextRetryDelay(uid types.UID, servedFor time.Duration) time.Duration {
	state, ok := m.retry[uid]
	if !ok {
		state = &retryState{}
		m.retry[uid] = state
	}

	var delay time.Duration
	if servedFor >= healthyRunThreshold {
		state.failures = 0
	} else {
		shift := state.failures
		if state.failures < maxRetryShift {
			state.failures++
		}
		delay = baseRetryDelay << shift
		if delay > maxRetryDelay {
			delay = maxRetryDelay
		}
	}
	state.nextStart = m.clock.Now().Add(delay)
	return delay
}

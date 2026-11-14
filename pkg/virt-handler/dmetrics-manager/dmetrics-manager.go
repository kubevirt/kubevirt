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

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/downwardmetrics"
	virtioserial "kubevirt.io/kubevirt/pkg/downwardmetrics/virtio-serial"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

func NewDownwardMetricsManager(nodeName string, recorder record.EventRecorder) *DownwardMetricsManager {
	return &DownwardMetricsManager{
		done:       false,
		nodeName:   nodeName,
		recorder:   recorder,
		stopServer: make(map[types.UID]context.CancelFunc),
	}
}

// DownwardMetricsManager controls the lifetime of the DownwardMetrics servers.
// Each server is tied to the lifetime of the VMI and DownwardMetricsManager itself.
type DownwardMetricsManager struct {
	lock       sync.Mutex
	done       bool
	nodeName   string
	recorder   record.EventRecorder
	stopServer map[types.UID]context.CancelFunc
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
}

// StopServer removes the VMI name from the list of served VMs a
// nd stop the DownwardMetrics server if necessary
func (m *DownwardMetricsManager) StopServer(vmi *v1.VirtualMachineInstance) {
	if !downwardmetrics.HasDevice(&vmi.Spec) {
		return
	}

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
}

// StartServer start a new DownwardMetrics server if the VM request it and is not already started
func (m *DownwardMetricsManager) StartServer(vmi *v1.VirtualMachineInstance, pid int) error {
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

	launcherSocketPath, err := cmdclient.FindSocket(vmi)
	if err != nil {
		return fmt.Errorf("failed to get the launcher socket for VMI [%s], error: %v", vmi.GetName(), err)
	}

	channelPath := downwardmetrics.ChannelSocketPathOnHost(pid)
	ctx, cancelCtx := context.WithCancel(context.Background())
	serverResult := virtioserial.RunDownwardMetricsVirtioServer(ctx, m.nodeName, channelPath, launcherSocketPath)
	m.stopServer[vmi.UID] = cancelCtx
	go m.watchServer(vmi, serverResult)

	return nil
}

// watchServer waits for the DownwardMetrics server of a VMI to terminate.
// If the server terminated because the virtio-serial connection could not be
// established, the VMI is deregistered, so that a later sync retries starting
// the server, and the lasting failure is surfaced as an event on the VMI.
func (m *DownwardMetricsManager) watchServer(vmi *v1.VirtualMachineInstance, serverResult <-chan error) {
	err := <-serverResult

	m.lock.Lock()
	delete(m.stopServer, vmi.UID)
	m.lock.Unlock()

	if err != nil {
		m.recorder.Eventf(
			vmi,
			k8sv1.EventTypeWarning,
			"DownwardMetricsUnavailable",
			"DownwardMetrics is unavailable for VMI [%s]: the virtio-serial server could not connect: %v",
			vmi.GetName(),
			err,
		)
	}
}

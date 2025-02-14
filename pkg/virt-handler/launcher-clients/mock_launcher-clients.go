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

package launcher_clients

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	virtcache "kubevirt.io/kubevirt/pkg/virt-handler/cache"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

type MockLauncherClientManager struct {
	Client            cmdclient.LauncherClient
	ClientInfo        *virtcache.LauncherClientInfo
	UnResponsive      bool
	Initialized       bool
	UnResponsiveError error
}

func (m *MockLauncherClientManager) GetVerifiedLauncherClient(vmi *v1.VirtualMachineInstance) (client cmdclient.LauncherClient, err error) {
	if m.Client != nil {
		return m.Client, nil
	}
	return nil, fmt.Errorf("Unknown client")
}

func (m *MockLauncherClientManager) GetLauncherClient(vmi *v1.VirtualMachineInstance) (cmdclient.LauncherClient, error) {
	return m.GetVerifiedLauncherClient(vmi)
}

func (m *MockLauncherClientManager) GetLauncherClientInfo(vmi *v1.VirtualMachineInstance) *virtcache.LauncherClientInfo {
	return m.ClientInfo
}

func (m *MockLauncherClientManager) CloseLauncherClient(vmi *v1.VirtualMachineInstance) error {
	return virtcache.GhostRecordGlobalStore.Delete(vmi.Namespace, vmi.Name)
}

func (m *MockLauncherClientManager) IsLauncherClientUnresponsive(vmi *v1.VirtualMachineInstance) (unresponsive bool, initialized bool, err error) {
	return m.UnResponsive, m.Initialized, m.UnResponsiveError
}

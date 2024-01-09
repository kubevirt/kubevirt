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
 * Copyright 2023 The KubeVirt Authors.
 *
 */
package cmdclient

import v1 "kubevirt.io/api/core/v1"

type ClientManager interface {
	Create(vmi *v1.VirtualMachineInstance) (LauncherClient, error)
	Socket(vmi *v1.VirtualMachineInstance) (string, error)
	IsNotResponsive(socketPath string) bool
}

type clientManager struct{}

func NewClientManager() *clientManager {
	return &clientManager{}
}

func (cm *clientManager) Create(vmi *v1.VirtualMachineInstance) (LauncherClient, error) {
	socketPath, err := FindSocketOnHost(vmi)
	if err != nil {
		return nil, err
	}

	return NewClient(socketPath)
}

func (cm *clientManager) Socket(vmi *v1.VirtualMachineInstance) (string, error) {
	return FindSocketOnHost(vmi)
}

func (cm *clientManager) IsNotResponsive(socketPath string) bool {
	return IsSocketUnresponsive(socketPath)
}

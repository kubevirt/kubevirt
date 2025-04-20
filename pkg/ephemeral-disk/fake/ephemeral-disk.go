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
 */

package fake

import (
	"path/filepath"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type MockEphemeralDiskImageCreator struct {
	BaseDir string
}

func (m *MockEphemeralDiskImageCreator) CreateBackedImageForVolume(_ v1.Volume, _ string, _ string) error {
	return nil
}

func (m *MockEphemeralDiskImageCreator) CreateEphemeralImages(_ *v1.VirtualMachineInstance, _ *api.Domain) error {
	return nil
}

func (m *MockEphemeralDiskImageCreator) GetFilePath(volumeName string) string {
	return filepath.Join(m.BaseDir, volumeName, "disk.qcow2")
}

func (m *MockEphemeralDiskImageCreator) Init() error {
	return nil
}

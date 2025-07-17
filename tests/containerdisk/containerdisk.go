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

package tests

import (
	"fmt"

	"kubevirt.io/kubevirt/tests/flags"
)

type ContainerDisk string

const (
	ContainerDiskCirrosCustomLocation ContainerDisk = "cirros-custom"
	ContainerDiskAlpineCustomLocation ContainerDisk = "alpine-custom"
	ContainerDiskCirros               ContainerDisk = "cirros"
	ContainerDiskAlpine               ContainerDisk = "alpine"
	ContainerDiskAlpineTestTooling    ContainerDisk = "alpine-with-test-tooling"
	ContainerDiskFedoraTestTooling    ContainerDisk = "fedora-with-test-tooling"
	ContainerDiskVirtio               ContainerDisk = "virtio-container-disk"
	ContainerDiskEmpty                ContainerDisk = "empty"
	ContainerDiskFedoraRealtime       ContainerDisk = "fedora-realtime"
	KernelBoot                        ContainerDisk = "alpine-ext-kernel-boot-demo"
)

const (
	FedoraVolumeSize = "6Gi"
	CirrosVolumeSize = "512Mi"
	AlpineVolumeSize = "512Mi"
	BlankVolumeSize  = "16Mi"
	VirtioVolumeSize = "750Mi"
)

// ContainerDiskFor takes the name of an image and returns the full
// registry diks image path.
// Use the ContainerDisk* constants as input values.
func ContainerDiskFor(name ContainerDisk) string {
	return ContainerDiskFromRegistryFor(flags.KubeVirtUtilityRepoPrefix, name)
}

func DataVolumeImportUrlForContainerDisk(name ContainerDisk) string {
	return DataVolumeImportUrlFromRegistryForContainerDisk(flags.KubeVirtUtilityRepoPrefix, name)
}

func DataVolumeImportUrlFromRegistryForContainerDisk(registry string, name ContainerDisk) string {
	return fmt.Sprintf("docker://%s", ContainerDiskFromRegistryFor(registry, name))
}

func ContainerDiskFromRegistryFor(registry string, name ContainerDisk) string {
	switch name {
	case ContainerDiskCirros, ContainerDiskAlpine, ContainerDiskCirrosCustomLocation, ContainerDiskAlpineCustomLocation:
		return fmt.Sprintf("%s/%s-container-disk-demo:%s", registry, name, flags.KubeVirtUtilityVersionTag)
	case ContainerDiskVirtio:
		return fmt.Sprintf("%s/virtio-container-disk:%s", registry, flags.KubeVirtUtilityVersionTag)
	case ContainerDiskFedoraTestTooling, ContainerDiskFedoraRealtime, ContainerDiskAlpineTestTooling:
		return fmt.Sprintf("%s/%s-container-disk:%s", registry, name, flags.KubeVirtUtilityVersionTag)
	case KernelBoot:
		return fmt.Sprintf("%s/alpine-ext-kernel-boot-demo:%s", registry, flags.KubeVirtUtilityVersionTag)
	}

	panic(fmt.Sprintf("Unsupported registry disk %s", name))
}

func ContainerDiskSizeBySourceURL(url string) string {
	if url == DataVolumeImportUrlForContainerDisk(ContainerDiskFedoraTestTooling) ||
		url == DataVolumeImportUrlForContainerDisk(ContainerDiskFedoraRealtime) {
		return FedoraVolumeSize
	}

	return CirrosVolumeSize
}

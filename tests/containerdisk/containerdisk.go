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
 * Copyright 2020 Red Hat, Inc.
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
	ContainerDiskCirros               ContainerDisk = "cirros"
	ContainerDiskAlpine               ContainerDisk = "alpine"
	ContainerDiskFedora               ContainerDisk = "fedora-cloud"
	ContainerDiskFedoraSRIOVLane      ContainerDisk = "fedora-sriov-lane"
	ContainerDiskFedoraTestTooling    ContainerDisk = "fedora-with-test-tooling"
	ContainerDiskMicroLiveCD          ContainerDisk = "microlivecd"
	ContainerDiskVirtio               ContainerDisk = "virtio-container-disk"
	ContainerDiskEmpty                ContainerDisk = "empty"
)

// ContainerDiskFor takes the name of an image and returns the full
// registry diks image path.
// Use the ContainerDisk* constants as input values.
func ContainerDiskFor(name ContainerDisk) string {
	switch name {
	case ContainerDiskCirros, ContainerDiskAlpine, ContainerDiskFedora, ContainerDiskMicroLiveCD, ContainerDiskCirrosCustomLocation:
		return fmt.Sprintf("%s/%s-container-disk-demo:%s", flags.KubeVirtUtilityRepoPrefix, name, flags.KubeVirtUtilityVersionTag)
	case ContainerDiskVirtio:
		return fmt.Sprintf("%s/virtio-container-disk:%s", flags.KubeVirtUtilityRepoPrefix, flags.KubeVirtUtilityVersionTag)
	case ContainerDiskFedoraSRIOVLane, ContainerDiskFedoraTestTooling:
		return fmt.Sprintf("%s/%s-container-disk:%s", flags.KubeVirtUtilityRepoPrefix, name, flags.KubeVirtUtilityVersionTag)
	}
	panic(fmt.Sprintf("Unsupported registry disk %s", name))
}

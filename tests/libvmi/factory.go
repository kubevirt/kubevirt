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

package libvmi

import (
	kvirtv1 "kubevirt.io/client-go/api/v1"

	cd "kubevirt.io/kubevirt/tests/containerdisk"
)

// Default VMI values
const (
	DefaultTestGracePeriod int64 = 0
	DefaultVmiName               = "testvmi"
)

// NewFedora instantiates a new Fedora based VMI configuration,
// building its extra properties based on the specified With* options.
func NewFedora(opts ...Option) *kvirtv1.VirtualMachineInstance {
	userData := `#!/bin/bash
	echo "fedora" | passwd fedora --stdin
	echo `

	return newFedora(cd.ContainerDiskFedora, userData, opts...)
}

// NewSriovFedora instantiates a new Fedora based VMI configuration,
// building its extra properties based on the specified With* options, the
// image used include Guest Agent and some moduled needed by SRIOV.
func NewSriovFedora(opts ...Option) *kvirtv1.VirtualMachineInstance {
	userData := `#!/bin/bash
	setenforce 0
	rm -f /var/lib/cloud/instance/boot-finished
	echo "fedora" | passwd fedora --stdin`

	userData = addStartGuestAgentUserDataSuffix(userData)
	return newFedora(cd.ContainerDiskFedoraSRIOVLane, userData, opts...)
}

// Start Guest agent must be that last operation of UserData.
// This will allows to wait for Agent connected event and to know
// that applying UserData is finished, and that a login can proceed.
func addStartGuestAgentUserDataSuffix(userData string) string {
	userData += `
	cp /home/fedora/qemu-guest-agent.service /lib/systemd/system/
	systemctl daemon-reload
	systemctl start qemu-guest-agent
	systemctl enable qemu-guest-agent`

	return userData
}

// NewFedora instantiates a new Fedora based VMI configuration with specified
// containerDisk, building its extra properties based on the specified With*
// options.
func newFedora(containerDisk cd.ContainerDisk, userData string, opts ...Option) *kvirtv1.VirtualMachineInstance {
	fedoraOptions := []Option{
		WithTerminationGracePeriod(DefaultTestGracePeriod),
		WithResourceMemory("512M"),
		WithRng(),
		WithContainerImage(cd.ContainerDiskFor(containerDisk)),
		WithCloudInitNoCloudUserData(userData, false),
	}
	opts = append(fedoraOptions, opts...)
	return New(RandName(DefaultVmiName), opts...)
}

// NewCirros instantiates a new CirrOS based VMI configuration
func NewCirros(opts ...Option) *kvirtv1.VirtualMachineInstance {
	cirrosOpts := []Option{
		WithContainerImage(cd.ContainerDiskFor(cd.ContainerDiskCirros)),
		WithCloudInitNoCloudUserData("#!/bin/bash\necho 'hello'\n", true),
		WithResourceMemory("64M"),
		WithTerminationGracePeriod(DefaultTestGracePeriod),
	}
	cirrosOpts = append(cirrosOpts, opts...)
	return New(RandName(DefaultVmiName), cirrosOpts...)
}

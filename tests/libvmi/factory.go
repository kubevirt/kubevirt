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
	DefaultResourceMemory        = "8192Ki"
	DefaultTestGracePeriod int64 = 0
	DefaultVmiName               = "testvmi"
)

// NewFedora instantiates a new Fedora based VMI configuration,
// building its extra properties based on the specified With* options.
func NewFedora(opts ...Option) *kvirtv1.VirtualMachineInstance {
	configurePassword := `#!/bin/bash
	echo "fedora" |passwd fedora --stdin
	echo `

	fedoraOptions := append(
		defaultOptions(),
		WithResourceMemory("512M"),
		WithRng(),
		WithContainerImage(cd.ContainerDiskFor(cd.ContainerDiskFedora)),
		WithCloudInitNoCloudUserData(configurePassword, true),
	)
	opts = append(fedoraOptions, opts...)
	return New(RandName(DefaultVmiName), opts...)
}

// NewCirros instantiates a new CirrOS based VMI configuration
func NewCirros(opts ...Option) *kvirtv1.VirtualMachineInstance {
	cirrosOpts := []Option{
		WithContainerImage(cd.ContainerDiskFor(cd.ContainerDiskCirros)),
		WithCloudInitNoCloudUserData("#!/bin/bash\necho 'hello'\n", true),
		WithResourceMemory("128M"),
		WithTerminationGracePeriod(DefaultTestGracePeriod),
	}
	cirrosOpts = append(cirrosOpts, opts...)
	return New(RandName(DefaultVmiName), cirrosOpts...)
}

// defaultOptions returns a list of "default" options.
func defaultOptions() []Option {
	return []Option{
		WithInterface(InterfaceDeviceWithMasqueradeBinding()),
		WithNetwork(kvirtv1.DefaultPodNetwork()),
		WithTerminationGracePeriod(DefaultTestGracePeriod),
		WithResourceMemory(DefaultResourceMemory),
	}
}

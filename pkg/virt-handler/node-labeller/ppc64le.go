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

package nodelabeller

import "kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/util"

// Ensure that there is a compile error should the struct not implement the archLabeller interface anymore.
var _ = archLabeller(&archLabellerPPC64LE{})

type archLabellerPPC64LE struct{}

func (archLabellerPPC64LE) defaultVendor() string {
	// libvirt does not always report a CPU vendor for POWER hosts, and IBM is the only vendor of POWER CPUs.
	return "IBM"
}

func (archLabellerPPC64LE) requirePolicy(policy string) bool {
	return policy == util.RequirePolicy
}

func (archLabellerPPC64LE) hasHostSupportedFeatures() bool {
	// virsh hypervisor-cpu-baseline is only used on x86 and s390x, so no supported_features.xml exists on ppc64le.
	return false
}

func (archLabellerPPC64LE) supportsHostModel() bool {
	return true
}

func (archLabellerPPC64LE) supportsNamedModels() bool {
	return true
}

func (archLabellerPPC64LE) arch() string {
	return ppc64le
}

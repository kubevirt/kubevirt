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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package virt_manifest

import (
	"kubevirt.io/kubevirt/pkg/api/v1"
)

func AddMinimalVMSpec(vm *v1.VirtualMachine) {
	// Make sure the domain name matches the VM name
	if vm.Spec.Domain == nil {
		vm.Spec.Domain = new(v1.DomainSpec)
	}

	AddMinimalDomainSpec(vm.Spec.Domain)
}

func AddMinimalDomainSpec(dom *v1.DomainSpec) {
	for idx, graphics := range dom.Devices.Graphics {
		if graphics.Type == "spice" {
			if graphics.Listen.Type == "" {
				dom.Devices.Graphics[idx].Listen.Type = "address"
			}
			if ((graphics.Listen.Type == "address") ||
				(graphics.Listen.Type == "")) &&
				(graphics.Listen.Address == "") {
				dom.Devices.Graphics[idx].Listen.Address = "0.0.0.0"
			}
		}
	}
}

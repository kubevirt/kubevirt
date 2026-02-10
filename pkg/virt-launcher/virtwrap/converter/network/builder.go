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

package network

import "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

type builderOption func(p *api.Interface)

func newDomainInterface(name, modelType string, options ...builderOption) api.Interface {
	iface := api.Interface{
		Alias: api.NewUserDefinedAlias(name),
		Model: &api.Model{Type: modelType},
	}

	for _, f := range options {
		f(&iface)
	}

	return iface
}

func withDriver(driver *api.InterfaceDriver) builderOption {
	return func(iface *api.Interface) {
		iface.Driver = driver
	}
}

func withPCIAddress(pciAddress *api.Address) builderOption {
	return func(iface *api.Interface) {
		iface.Address = pciAddress
	}
}

func withACPIIndex(acpiIndex uint) builderOption {
	return func(iface *api.Interface) {
		iface.ACPI = &api.ACPI{Index: acpiIndex}
	}
}

func withIfaceType(ifaceType string) builderOption {
	return func(iface *api.Interface) {
		iface.Type = ifaceType
	}
}

func withBootOrder(bootOrder uint) builderOption {
	return func(iface *api.Interface) {
		iface.BootOrder = &api.BootOrder{Order: bootOrder}
	}
}

func withROMDisabled() builderOption {
	return func(iface *api.Interface) {
		iface.Rom = &api.Rom{Enabled: "no"}
	}
}

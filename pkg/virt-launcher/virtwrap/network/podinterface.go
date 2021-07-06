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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package network

import (
	"fmt"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type LibvirtSpecGenerator interface {
	generate(domainIface api.Interface) error
}

func newMacvtapLibvirtSpecGenerator(iface *v1.Interface, domain *api.Domain) *MacvtapLibvirtSpecGenerator {
	return &MacvtapLibvirtSpecGenerator{
		vmiSpecIface: iface,
		domain:       domain,
	}
}

func newMasqueradeLibvirtSpecGenerator(iface *v1.Interface, domain *api.Domain) *MasqueradeLibvirtSpecGenerator {
	return &MasqueradeLibvirtSpecGenerator{
		vmiSpecIface: iface,
		domain:       domain,
	}
}

func newSlirpLibvirtSpecGenerator(iface *v1.Interface, domain *api.Domain) *SlirpLibvirtSpecGenerator {
	return &SlirpLibvirtSpecGenerator{
		vmiSpecIface: iface,
		domain:       domain,
	}
}

func newBridgeLibvirtSpecGenerator(iface *v1.Interface, domain *api.Domain) *BridgeLibvirtSpecGenerator {
	return &BridgeLibvirtSpecGenerator{
		vmiSpecIface: iface,
		domain:       domain,
	}
}

type BridgeLibvirtSpecGenerator struct {
	vmiSpecIface *v1.Interface
	domain       *api.Domain
}

func (b *BridgeLibvirtSpecGenerator) generate(domainIface api.Interface) error {
	ifaces := b.domain.Spec.Devices.Interfaces
	for i, iface := range ifaces {
		if iface.Alias.GetName() == b.vmiSpecIface.Name {
			ifaces[i].MTU = domainIface.MTU
			ifaces[i].MAC = domainIface.MAC
			ifaces[i].Target = domainIface.Target
			break
		}
	}
	return nil
}

type MasqueradeLibvirtSpecGenerator struct {
	vmiSpecIface *v1.Interface
	domain       *api.Domain
}

func (b *MasqueradeLibvirtSpecGenerator) generate(domainIface api.Interface) error {
	ifaces := b.domain.Spec.Devices.Interfaces
	for i, iface := range ifaces {
		if iface.Alias.GetName() == b.vmiSpecIface.Name {
			ifaces[i].MTU = domainIface.MTU
			ifaces[i].MAC = domainIface.MAC
			ifaces[i].Target = domainIface.Target
			break
		}
	}
	return nil
}

type SlirpLibvirtSpecGenerator struct {
	vmiSpecIface *v1.Interface
	domain       *api.Domain
}

func (b *SlirpLibvirtSpecGenerator) generate(api.Interface) error {
	// remove slirp interface from domain spec devices interfaces
	var foundIfaceModelType string
	ifaces := b.domain.Spec.Devices.Interfaces
	for i, iface := range ifaces {
		if iface.Alias.GetName() == b.vmiSpecIface.Name {
			b.domain.Spec.Devices.Interfaces = append(ifaces[:i], ifaces[i+1:]...)
			foundIfaceModelType = iface.Model.Type
			break
		}
	}

	if foundIfaceModelType == "" {
		return fmt.Errorf("failed to find interface %s in vmi spec", b.vmiSpecIface.Name)
	}

	qemuArg := fmt.Sprintf("%s,netdev=%s,id=%s", foundIfaceModelType, b.vmiSpecIface.Name, b.vmiSpecIface.Name)
	if b.vmiSpecIface.MacAddress != "" {
		// We assume address was already validated in API layer so just pass it to libvirt as-is.
		qemuArg += fmt.Sprintf(",mac=%s", b.vmiSpecIface.MacAddress)
	}
	// Add interface configuration to qemuArgs
	b.domain.Spec.QEMUCmd.QEMUArg = append(b.domain.Spec.QEMUCmd.QEMUArg, api.Arg{Value: "-device"})
	b.domain.Spec.QEMUCmd.QEMUArg = append(b.domain.Spec.QEMUCmd.QEMUArg, api.Arg{Value: qemuArg})

	return nil
}

type MacvtapLibvirtSpecGenerator struct {
	vmiSpecIface *v1.Interface
	domain       *api.Domain
}

func (b *MacvtapLibvirtSpecGenerator) generate(domainIface api.Interface) error {
	ifaces := b.domain.Spec.Devices.Interfaces
	for i, iface := range ifaces {
		if iface.Alias.GetName() == b.vmiSpecIface.Name {
			ifaces[i].MTU = domainIface.MTU
			ifaces[i].MAC = domainIface.MAC
			ifaces[i].Target = domainIface.Target
			break
		}
	}
	return nil
}

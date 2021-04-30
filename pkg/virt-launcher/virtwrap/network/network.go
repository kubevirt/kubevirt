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
	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const primaryPodInterfaceName = "eth0"

// NetworkingConfigurator is responsible for extending the pod network into the
// Virtual Machines.
type NetworkingConfigurator interface {
	Setup() error
}

// InfraNetworkingConfigurator creates and configures networking infrastructure
// for the VMI in the virt-launcher's network namespace, which is reached via
// the PID.
type InfraNetworkingConfigurator struct {
	vmi          *v1.VirtualMachineInstance
	launcherPID  int
	NicGenerator
}

// NetworkingSpecGenerator generates the required libvirt Dom XML for the
// desired virtual machine.
type NetworkingSpecGenerator struct {
	vmi          *v1.VirtualMachineInstance
	domXML       api.Domain
	NicGenerator
}

// NicGenerator generates the required podNIC structs, which are responsible
// for extending networking from the pod into the Virtual Machine.
type NicGenerator struct {
	vmi          *v1.VirtualMachineInstance
	handler      netdriver.NetworkHandler
	cacheFactory cache.InterfaceCacheFactory
}

func (v NicGenerator) getNICs(launcherPID *int) ([]podNIC, error) {
	nics := []podNIC{}

	if len(v.vmi.Spec.Domain.Devices.Interfaces) == 0 {
		return nics, nil
	}

	for i, _ := range v.vmi.Spec.Networks {
		nic, err := newPodNIC(v.vmi, &v.vmi.Spec.Networks[i], v.handler, v.cacheFactory, launcherPID)
		if err != nil {
			return nil, err
		}
		nics = append(nics, *nic)
	}
	return nics, nil
}

// NewInfraNetworkingConfigurator returns a InfraNetworkingConfigurator.
func NewInfraNetworkingConfigurator(vmi *v1.VirtualMachineInstance, cacheFactory cache.InterfaceCacheFactory, launcherPID int) InfraNetworkingConfigurator {
	networkDriver := &netdriver.NetworkUtilsHandler{}
	return InfraNetworkingConfigurator{
		vmi:          vmi,
		launcherPID:  launcherPID,
		NicGenerator: newNicGeneratorWithHandlerAndCache(vmi, networkDriver, cacheFactory),
	}
}

// Setup will create the auxiliary networking infrastructure to enxtend the
// pod networking into the Virtual Machine.
func (v InfraNetworkingConfigurator) Setup() error {
	nics, err := v.getNICs(&v.launcherPID)
	if err != nil {
		return nil
	}
	for _, nic := range nics {
		if err := nic.SetupNetworkInfrastructure(); err != nil {
			return fmt.Errorf("failed plugging phase1 at nic '%s': %w", nic.podInterfaceName, err)
		}
	}
	return nil
}

// NewNetworkingSpecGenerator returns a NetworkingSpecGenerator.
func NewNetworkingSpecGenerator(vmi *v1.VirtualMachineInstance, cacheFactory cache.InterfaceCacheFactory, domain api.Domain) NetworkingSpecGenerator {
	networkDriver := &netdriver.NetworkUtilsHandler{}
	return NetworkingSpecGenerator{
		vmi:          vmi,
		domXML:       domain,
		NicGenerator: newNicGeneratorWithHandlerAndCache(vmi, networkDriver, cacheFactory),
	}
}

// Setup will decorate the DOM XML Devices.Interfaces with the data Libvirt
// requires to extend networking from the pod into the Virtual Machine.
func (v NetworkingSpecGenerator) Setup() error {
	nics, err := v.getNICs(nil)
	if err != nil {
		return nil
	}
	for _, nic := range nics {
		if err := nic.UnpriviligedSetup(&v.domXML); err != nil {
			return fmt.Errorf("failed plugging phase1 at nic '%s': %w", nic.podInterfaceName, err)
		}
	}
	return nil
}

func newNicGeneratorWithHandlerAndCache(vmi *v1.VirtualMachineInstance, handler netdriver.NetworkHandler, cacheFactory cache.InterfaceCacheFactory) NicGenerator {
	return NicGenerator{
		vmi:          vmi,
		handler:      handler,
		cacheFactory: cacheFactory,
	}
}

func newNicGenerator(vmi *v1.VirtualMachineInstance, cacheFactory cache.InterfaceCacheFactory) NicGenerator {
	return newNicGeneratorWithHandlerAndCache(vmi, &netdriver.NetworkUtilsHandler{}, cacheFactory)
}

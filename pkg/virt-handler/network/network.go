/*
 * This file is part of the kubevirt project
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

package network

import (
	"errors"

	"github.com/jeevatkm/go-model"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-handler/network/cniproxy"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
)

const HostNetNS = "/proc/1/ns/net"

type VirtualInterface interface {
	SetInterfaceAttributes(mac string, ip string, device string) error
	Plug(domainManager virtwrap.DomainManager) error
	Unplug(domainManager virtwrap.DomainManager) error
	GetConfig() (*v1.Interface, error)
	DecorateInterfaceMetadata() *v1.MetadataDevice
}

func GetInterfaceType(objName string) (VirtualInterface, error) {
	switch objName {
	//TODO:(vladikr) We can extend this to other types
	case "PodNetworking":
		return new(PodNetworkInterface), nil
	default:
		return nil, errors.New(objName + " is not a plugable interface type.")
	}
}

func UnPlugNetworkDevices(vm *v1.VirtualMachine, domainManager virtwrap.DomainManager) error {
	if vm.Spec.Domain.Metadata != nil {
		for _, inter := range vm.Spec.Domain.Metadata.Interfaces.Devices {
			log.Log.Debugf("unplugging interface: %s, type: %s, from VM: %s", inter.Device, inter.Type, vm.ObjectMeta.Name)
			vif, err := GetInterfaceType(inter.Type)
			if err != nil {
				continue
			}
			vif.SetInterfaceAttributes(inter.Mac, inter.IP, inter.Device)
			if err != nil {
				return err
			}
			err = vif.Unplug(domainManager)
			if err != nil {
				log.Log.Reason(err).Warningf("failed to unplug: ", inter.Device, "for VM: ", vm.ObjectMeta.Name)
			}

		}
	}
	return nil
}

func PlugNetworkDevices(vm *v1.VirtualMachine, domainManager virtwrap.DomainManager) (*v1.VirtualMachine, error) {
	vmCopy := &v1.VirtualMachine{}
	model.Copy(vmCopy, vm)

	//TODO:(vladikr) Currently we support only one interface per vm. Improve this once we'll start supporting more.
	for idx, inter := range vmCopy.Spec.Domain.Devices.Interfaces {
			vif, err := GetInterfaceType(inter.Type)
			if err != nil {
				continue
			}
			err = vif.Plug(domainManager)
			if err != nil {
				return nil, err
			}

			// Add VIF to VM config
			ifconf, err := vif.GetConfig()
			if err != nil {
				log.Log.Reason(err).Error("failed to get VIF config.")
				return nil, err
			}
			vmCopy.Spec.Domain.Devices.Interfaces[idx] = *ifconf
			ifaceMeta := vif.DecorateInterfaceMetadata()
			if vmCopy.Spec.Domain.Metadata == nil {
				vmCopy.Spec.Domain.Metadata = &v1.Metadata{}
			}
			vmCopy.Spec.Domain.Metadata.Interfaces.Devices = append(vmCopy.Spec.Domain.Metadata.Interfaces.Devices, *ifaceMeta)
	}

	return vmCopy, nil
}

func GetContainerInteface(iface string) (*cniproxy.CNIProxy, error) {
	runtime, err := cniproxy.BuildRuntimeConfig(iface)
	if err != nil {
		return nil, err
	}
	cniProxy, err := cniproxy.GetProxy(runtime)
	if err != nil {
		log.Log.Reason(err).Error("failed to get CNI Proxy")
		return nil, err
	}
	return cniProxy, nil
}

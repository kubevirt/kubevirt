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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package network

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/network/cache"
	netvmispec "kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type NetStat struct {
	ifaceCacheFactory cache.InterfaceCacheFactory

	// In memory cache, storing pod interface information.
	// key is the file path, value is the contents.
	// if key exists, then don't read directly from file.
	podInterfaceVolatileCache sync.Map
}

func NewNetStat(ifaceCacheFactory cache.InterfaceCacheFactory) *NetStat {
	return &NetStat{
		ifaceCacheFactory:         ifaceCacheFactory,
		podInterfaceVolatileCache: sync.Map{},
	}
}

func (c *NetStat) Teardown(vmi *v1.VirtualMachineInstance) {
	c.podInterfaceVolatileCache.Range(func(key, value interface{}) bool {
		if strings.HasPrefix(key.(string), string(vmi.UID)) {
			c.podInterfaceVolatileCache.Delete(key)
		}
		return true
	})
}

func (c *NetStat) PodInterfaceVolatileDataIsCached(vmi *v1.VirtualMachineInstance, ifaceName string) bool {
	_, exists := c.podInterfaceVolatileCache.Load(vmiInterfaceKey(vmi.UID, ifaceName))
	return exists
}

func (c *NetStat) CachePodInterfaceVolatileData(vmi *v1.VirtualMachineInstance, ifaceName string, data *cache.PodCacheInterface) {
	c.podInterfaceVolatileCache.Store(vmiInterfaceKey(vmi.UID, ifaceName), data)
}

func (c *NetStat) UpdateStatus(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if domain == nil {
		return nil
	}

	interfacesStatus, err := c.ifacesStatusFromPod(vmi)
	if err != nil {
		return err
	}
	vmi.Status.Interfaces = interfacesStatus

	if len(domain.Spec.Devices.Interfaces) > 0 || len(domain.Status.Interfaces) > 0 {
		// This calculates the vmi.Status.Interfaces based on the following data sets:
		// - vmi.Status.Interfaces - interfaces data (IP/s) collected from the pod (non-volatile cache).
		// - domain.Spec - interfaces form the Spec
		// - domain.Status.Interfaces - interfaces reported by guest agent (empty if Qemu agent not running)
		newInterfaces := []v1.VirtualMachineInstanceNetworkInterface{}

		existingInterfaceStatusByName := map[string]v1.VirtualMachineInstanceNetworkInterface{}
		for _, existingInterfaceStatus := range vmi.Status.Interfaces {
			if existingInterfaceStatus.Name != "" {
				existingInterfaceStatusByName[existingInterfaceStatus.Name] = existingInterfaceStatus
			}
		}

		domainInterfaceStatusByMac := map[string]api.InterfaceStatus{}
		for _, domainInterfaceStatus := range domain.Status.Interfaces {
			domainInterfaceStatusByMac[domainInterfaceStatus.Mac] = domainInterfaceStatus
		}

		existingInterfacesSpecByName := map[string]v1.Interface{}
		for _, existingInterfaceSpec := range vmi.Spec.Domain.Devices.Interfaces {
			existingInterfacesSpecByName[existingInterfaceSpec.Name] = existingInterfaceSpec
		}
		existingNetworksByName := map[string]v1.Network{}
		for _, existingNetwork := range vmi.Spec.Networks {
			existingNetworksByName[existingNetwork.Name] = existingNetwork
		}

		// Iterate through all domain.Spec interfaces
		for _, domainInterface := range domain.Spec.Devices.Interfaces {
			interfaceMAC := domainInterface.MAC.MAC
			var newInterface v1.VirtualMachineInstanceNetworkInterface
			var isForwardingBindingInterface = false

			if existingInterfacesSpecByName[domainInterface.Alias.GetName()].Masquerade != nil || existingInterfacesSpecByName[domainInterface.Alias.GetName()].Slirp != nil {
				isForwardingBindingInterface = true
			}

			if existingInterface, exists := existingInterfaceStatusByName[domainInterface.Alias.GetName()]; exists {
				// Reuse previously calculated interface from vmi.Status.Interfaces, updating the MAC from domain.Spec
				// Only interfaces defined in domain.Spec are handled here
				newInterface = existingInterface
				newInterface.MAC = interfaceMAC

				// If it is a Combination of Masquerade+Pod network, check IP from file cache
				if existingInterfacesSpecByName[domainInterface.Alias.GetName()].Masquerade != nil && existingNetworksByName[domainInterface.Alias.GetName()].NetworkSource.Pod != nil {
					iface, err := c.getPodInterfacefromFileCache(vmi, domainInterface.Alias.GetName())
					if err != nil {
						return err
					}

					if !reflect.DeepEqual(iface.PodIPs, existingInterfaceStatusByName[domainInterface.Alias.GetName()].IPs) {
						newInterface.Name = domainInterface.Alias.GetName()
						newInterface.IP = iface.PodIP
						newInterface.IPs = iface.PodIPs
					}
				}
			} else {
				// If not present in vmi.Status.Interfaces, create a new one based on domain.Spec
				newInterface = v1.VirtualMachineInstanceNetworkInterface{
					MAC:  interfaceMAC,
					Name: domainInterface.Alias.GetName(),
				}
			}

			// Update IP info based on information from domain.Status.Interfaces (Qemu guest)
			// Remove the interface from domainInterfaceStatusByMac to mark it as handled
			if interfaceStatus, exists := domainInterfaceStatusByMac[interfaceMAC]; exists {
				newInterface.InterfaceName = interfaceStatus.InterfaceName
				newInterface.InfoSource = interfaceStatus.InfoSource
				// Do not update if interface has Masquerede binding
				// virt-controller should update VMI status interface with Pod IP instead
				if !isForwardingBindingInterface {
					newInterface.IP = interfaceStatus.Ip
					newInterface.IPs = interfaceStatus.IPs
				}
				delete(domainInterfaceStatusByMac, interfaceMAC)
			} else {
				newInterface.InfoSource = netvmispec.InfoSourceDomain
			}

			newInterfaces = append(newInterfaces, newInterface)
		}

		// If any of domain.Status.Interfaces were not handled above, it means that the vm contains additional
		// interfaces not defined in domain.Spec.Devices.Interfaces (most likely added by user on VM or a SRIOV interface)
		// Add them to vmi.Status.Interfaces
		setMissingSRIOVInterfacesNames(existingInterfacesSpecByName, domainInterfaceStatusByMac)
		for interfaceMAC, domainInterfaceStatus := range domainInterfaceStatusByMac {
			newInterface := v1.VirtualMachineInstanceNetworkInterface{
				Name:          domainInterfaceStatus.Name,
				MAC:           interfaceMAC,
				IP:            domainInterfaceStatus.Ip,
				IPs:           domainInterfaceStatus.IPs,
				InterfaceName: domainInterfaceStatus.InterfaceName,
				InfoSource:    domainInterfaceStatus.InfoSource,
			}
			newInterfaces = append(newInterfaces, newInterface)
		}
		vmi.Status.Interfaces = newInterfaces
	}
	return nil
}

func (c *NetStat) ifacesStatusFromPod(vmi *v1.VirtualMachineInstance) ([]v1.VirtualMachineInstanceNetworkInterface, error) {
	ifacesStatus := []v1.VirtualMachineInstanceNetworkInterface{}
	for _, network := range vmi.Spec.Networks {
		podIface, err := c.getPodInterfacefromFileCache(vmi, network.Name)
		if err != nil {
			return nil, err
		}

		ifacesStatus = append(ifacesStatus, v1.VirtualMachineInstanceNetworkInterface{
			Name: network.Name,
			IP:   podIface.PodIP,
			IPs:  podIface.PodIPs,
		})
	}
	return ifacesStatus, nil
}

func (c *NetStat) getPodInterfacefromFileCache(vmi *v1.VirtualMachineInstance, ifaceName string) (*cache.PodCacheInterface, error) {
	// Once the Interface files are set on the handler, they don't change
	// If already present in the map, don't read again
	cacheData, exists := c.podInterfaceVolatileCache.Load(vmiInterfaceKey(vmi.UID, ifaceName))
	if exists {
		return cacheData.(*cache.PodCacheInterface), nil
	}

	//FIXME error handling?
	podInterface, _ := c.ifaceCacheFactory.CacheForVMI(vmi).Read(ifaceName)

	c.podInterfaceVolatileCache.Store(vmiInterfaceKey(vmi.UID, ifaceName), podInterface)

	return podInterface, nil
}

func setMissingSRIOVInterfacesNames(interfacesSpecByName map[string]v1.Interface, interfacesStatusByMac map[string]api.InterfaceStatus) {
	for name, ifaceSpec := range interfacesSpecByName {
		if ifaceSpec.SRIOV == nil || ifaceSpec.MacAddress == "" {
			continue
		}
		if domainIfaceStatus, exists := interfacesStatusByMac[ifaceSpec.MacAddress]; exists {
			domainIfaceStatus.Name = name
			interfacesStatusByMac[ifaceSpec.MacAddress] = domainIfaceStatus
		}
	}
}

func vmiInterfaceKey(vmiUID types.UID, interfaceName string) string {
	return fmt.Sprintf("%s/%s", vmiUID, interfaceName)
}

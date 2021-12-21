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

// UpdateStatus calculates the vmi.Status.Interfaces based on the following data sets:
// - Pod interface cache: interfaces data (IP/s) collected from the cache (which was populated during the network setup).
// - domain.Spec: interfaces configuration as seen by the (libvirt) domain.
// - domain.Status.Interfaces: interfaces reported by the guest agent (empty if Qemu agent not running).
func (c *NetStat) UpdateStatus(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if domain == nil {
		return nil
	}

	interfacesStatus, err := c.ifacesStatusFromPodCache(vmi)
	if err != nil {
		return err
	}

	if len(domain.Spec.Devices.Interfaces) > 0 {
		interfacesStatus = ifacesStatusFromDomain(interfacesStatus, domain.Spec.Devices.Interfaces)
	}

	if len(domain.Status.Interfaces) > 0 {
		interfacesStatus = ifacesStatusFromGuestAgent(interfacesStatus, vmi.Spec.Domain.Devices.Interfaces, domain.Status.Interfaces)
	}

	vmi.Status.Interfaces = interfacesStatus
	return nil
}

func (c *NetStat) ifacesStatusFromPodCache(vmi *v1.VirtualMachineInstance) ([]v1.VirtualMachineInstanceNetworkInterface, error) {
	ifacesStatus := []v1.VirtualMachineInstanceNetworkInterface{}
	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		if iface.SRIOV != nil {
			continue
		}
		podIface, err := c.getPodInterfacefromFileCache(vmi, iface.Name)
		if err != nil {
			return nil, err
		}

		ifacesStatus = append(ifacesStatus, v1.VirtualMachineInstanceNetworkInterface{
			Name: iface.Name,
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
	podInterface, _ := c.ifaceCacheFactory.CacheForVMI(string(vmi.UID)).Read(ifaceName)

	c.podInterfaceVolatileCache.Store(vmiInterfaceKey(vmi.UID, ifaceName), podInterface)

	return podInterface, nil
}

func vmiInterfaceKey(vmiUID types.UID, interfaceName string) string {
	return fmt.Sprintf("%s/%s", vmiUID, interfaceName)
}

func ifacesStatusFromDomain(vmiStatusIfaces []v1.VirtualMachineInstanceNetworkInterface, domainSpecIfaces []api.Interface) []v1.VirtualMachineInstanceNetworkInterface {
	for _, domainSpecInterface := range domainSpecIfaces {
		interfaceMAC := domainSpecInterface.MAC.MAC
		domainIfaceAlias := domainSpecInterface.Alias.GetName()

		if interfaceStatus := netvmispec.LookupInterfaceStatusByName(vmiStatusIfaces, domainIfaceAlias); interfaceStatus != nil {
			interfaceStatus.MAC = interfaceMAC
			interfaceStatus.InfoSource = netvmispec.InfoSourceDomain
		} else {
			networkInterface := v1.VirtualMachineInstanceNetworkInterface{
				Name:       domainIfaceAlias,
				MAC:        interfaceMAC,
				InfoSource: netvmispec.InfoSourceDomain,
			}
			vmiStatusIfaces = append(vmiStatusIfaces, networkInterface)
		}
	}
	return vmiStatusIfaces
}

func ifacesStatusFromGuestAgent(vmiIfacesStatus []v1.VirtualMachineInstanceNetworkInterface, vmiIfacesSpec []v1.Interface, guestAgentInterfaces []api.InterfaceStatus) []v1.VirtualMachineInstanceNetworkInterface {
	vmiInterfacesSpecByName := netvmispec.IndexInterfaceSpecByName(vmiIfacesSpec)
	vmiInterfacesSpecByMac := netvmispec.IndexInterfaceSpecByMac(vmiIfacesSpec)

	for _, guestAgentInterface := range guestAgentInterfaces {
		if vmiIfaceStatus := netvmispec.LookupInterfaceStatusByMac(vmiIfacesStatus, guestAgentInterface.Mac); vmiIfaceStatus != nil {
			updateVMIIfaceStatusWithGuestAgentData(vmiIfaceStatus, guestAgentInterface, vmiInterfacesSpecByName)
		} else {
			setMissingSRIOVInterfaceName(&guestAgentInterface, vmiInterfacesSpecByMac)
			data := newVMIIfaceStatusFromGuestAgentData(guestAgentInterface)
			vmiIfacesStatus = append(vmiIfacesStatus, data)
		}
	}
	return vmiIfacesStatus
}

func updateVMIIfaceStatusWithGuestAgentData(ifaceStatus *v1.VirtualMachineInstanceNetworkInterface, guestAgentIface api.InterfaceStatus, ifacesSpecByName map[string]v1.Interface) {
	ifaceStatus.InterfaceName = guestAgentIface.InterfaceName
	ifaceStatus.InfoSource = guestAgentIface.InfoSource

	// Update IP/s only if the interface has no Masquerade/Slirp binding
	vmiSpecIface := ifacesSpecByName[ifaceStatus.Name]
	if vmiSpecIface.Masquerade == nil && vmiSpecIface.Slirp == nil {
		ifaceStatus.IP = guestAgentIface.Ip
		ifaceStatus.IPs = guestAgentIface.IPs
	}
}

func setMissingSRIOVInterfaceName(guestAgentInterface *api.InterfaceStatus, vmiInterfacesSpecByMac map[string]v1.Interface) {
	vmiSpecIface := vmiInterfacesSpecByMac[guestAgentInterface.Mac]
	if vmiSpecIface.SRIOV != nil {
		guestAgentInterface.Name = vmiSpecIface.Name
	}
}

func newVMIIfaceStatusFromGuestAgentData(guestAgentInterface api.InterfaceStatus) v1.VirtualMachineInstanceNetworkInterface {
	return v1.VirtualMachineInstanceNetworkInterface{
		Name:          guestAgentInterface.Name,
		MAC:           guestAgentInterface.Mac,
		IP:            guestAgentInterface.Ip,
		IPs:           guestAgentInterface.IPs,
		InterfaceName: guestAgentInterface.InterfaceName,
		InfoSource:    guestAgentInterface.InfoSource,
	}
}

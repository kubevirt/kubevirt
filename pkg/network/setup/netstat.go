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
	"kubevirt.io/kubevirt/pkg/network/sriov"
	netvmispec "kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type NetStat struct {
	cacheCreator cacheCreator

	// In memory cache, storing pod interface information.
	// key is the file path, value is the contents.
	// if key exists, then don't read directly from file.
	podInterfaceVolatileCache sync.Map
}

func NewNetStat() *NetStat {
	return NewNetStateWithCustomFactory(cache.CacheCreator{})
}

func NewNetStateWithCustomFactory(cacheCreator cacheCreator) *NetStat {
	return &NetStat{
		cacheCreator:              cacheCreator,
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

func (c *NetStat) CachePodInterfaceVolatileData(vmi *v1.VirtualMachineInstance, ifaceName string, data *cache.PodIfaceCacheData) {
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

	vmiInterfacesSpecByName := netvmispec.IndexInterfaceSpecByName(vmi.Spec.Domain.Devices.Interfaces)

	interfacesStatus := ifacesStatusFromDomainInterfaces(domain.Spec.Devices.Interfaces)
	interfacesStatus = append(interfacesStatus,
		sriovIfacesStatusFromDomainHostDevices(domain.Spec.Devices.HostDevices, vmiInterfacesSpecByName)...,
	)

	var err error
	if len(domain.Status.Interfaces) > 0 {
		interfacesStatus = ifacesStatusFromGuestAgent(interfacesStatus, domain.Status.Interfaces)

		natedIfacesSpec := netvmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(i v1.Interface) bool {
			return i.Masquerade != nil || i.Slirp != nil
		})
		if interfacesStatus, err = c.updateIfacesStatusFromPodCache(interfacesStatus, natedIfacesSpec, vmi); err != nil {
			return err
		}
	} else {
		interfacesStatus, err = c.updateIfacesStatusFromPodCache(interfacesStatus, vmi.Spec.Domain.Devices.Interfaces, vmi)
		if err != nil {
			return err
		}
	}

	vmi.Status.Interfaces = interfacesStatus
	return nil
}

// updateIfacesStatusFromPodCache updates the provided interfaces statuses with data (IP/s) from the pod-cache.
func (c *NetStat) updateIfacesStatusFromPodCache(ifacesStatus []v1.VirtualMachineInstanceNetworkInterface, ifacesSpec []v1.Interface, vmi *v1.VirtualMachineInstance) ([]v1.VirtualMachineInstanceNetworkInterface, error) {
	for _, iface := range ifacesSpec {
		ifaceStatus := netvmispec.LookupInterfaceStatusByName(ifacesStatus, iface.Name)
		if ifaceStatus == nil {
			continue
		}

		podIface, err := c.getPodInterfacefromFileCache(vmi, iface.Name)
		if err != nil {
			return nil, err
		}

		ifaceStatus.IP = podIface.PodIP
		ifaceStatus.IPs = podIface.PodIPs
	}
	return ifacesStatus, nil
}

func (c *NetStat) getPodInterfacefromFileCache(vmi *v1.VirtualMachineInstance, ifaceName string) (*cache.PodIfaceCacheData, error) {
	// Once the Interface files are set on the handler, they don't change
	// If already present in the map, don't read again
	cacheData, exists := c.podInterfaceVolatileCache.Load(vmiInterfaceKey(vmi.UID, ifaceName))
	if exists {
		return cacheData.(*cache.PodIfaceCacheData), nil
	}

	podInterface := &cache.PodIfaceCacheData{}
	if data, err := cache.ReadPodInterfaceCache(c.cacheCreator, string(vmi.UID), ifaceName); err == nil {
		//FIXME error handling?
		podInterface = data
	}

	c.podInterfaceVolatileCache.Store(vmiInterfaceKey(vmi.UID, ifaceName), podInterface)

	return podInterface, nil
}

func vmiInterfaceKey(vmiUID types.UID, interfaceName string) string {
	return fmt.Sprintf("%s/%s", vmiUID, interfaceName)
}

func ifacesStatusFromDomainInterfaces(domainSpecIfaces []api.Interface) []v1.VirtualMachineInstanceNetworkInterface {
	var vmiStatusIfaces []v1.VirtualMachineInstanceNetworkInterface

	for _, domainSpecIface := range domainSpecIfaces {
		vmiStatusIfaces = append(vmiStatusIfaces, v1.VirtualMachineInstanceNetworkInterface{
			Name:       domainSpecIface.Alias.GetName(),
			MAC:        domainSpecIface.MAC.MAC,
			InfoSource: netvmispec.InfoSourceDomain,
		})
	}
	return vmiStatusIfaces
}

func sriovIfacesStatusFromDomainHostDevices(hostDevices []api.HostDevice, vmiIfacesSpecByName map[string]v1.Interface) []v1.VirtualMachineInstanceNetworkInterface {
	var vmiStatusIfaces []v1.VirtualMachineInstanceNetworkInterface

	for _, hostDevice := range filterHostDevicesByAlias(hostDevices, sriov.AliasPrefix) {
		vmiStatusIface := v1.VirtualMachineInstanceNetworkInterface{
			Name:       hostDevice.Alias.GetName()[len(sriov.AliasPrefix):],
			InfoSource: netvmispec.InfoSourceDomain,
		}
		if iface, exists := vmiIfacesSpecByName[vmiStatusIface.Name]; exists {
			vmiStatusIface.MAC = iface.MacAddress
		}
		vmiStatusIfaces = append(vmiStatusIfaces, vmiStatusIface)
	}
	return vmiStatusIfaces
}

func ifacesStatusFromGuestAgent(vmiIfacesStatus []v1.VirtualMachineInstanceNetworkInterface, guestAgentInterfaces []api.InterfaceStatus) []v1.VirtualMachineInstanceNetworkInterface {
	for _, guestAgentInterface := range guestAgentInterfaces {
		if vmiIfaceStatus := netvmispec.LookupInterfaceStatusByMac(vmiIfacesStatus, guestAgentInterface.Mac); vmiIfaceStatus != nil {
			updateVMIIfaceStatusWithGuestAgentData(vmiIfaceStatus, guestAgentInterface)
			if !isGuestAgentIfaceOriginatedFromOldVirtLauncher(guestAgentInterface) {
				vmiIfaceStatus.InfoSource = netvmispec.InfoSourceDomainAndGA
			}
		} else {
			newVMIIfaceStatus := newVMIIfaceStatusFromGuestAgentData(guestAgentInterface)
			newVMIIfaceStatus.InfoSource = netvmispec.InfoSourceGuestAgent
			vmiIfacesStatus = append(vmiIfacesStatus, newVMIIfaceStatus)
		}
	}
	return vmiIfacesStatus
}

// For backward compatability with older virt-launchers, apply this logic:
// - When the domain status `InterfaceName` field is set, the data originates from the guest-agent.
//   This is true for old virt-launchers and new ones alike.
// - When the domain status `InterfaceName` field is not set (empty), the data originates from a merge
//   at an old virt-launcher and comes from the domain spec. In such cases, no action is to be taken.
func isGuestAgentIfaceOriginatedFromOldVirtLauncher(guestAgentInterface api.InterfaceStatus) bool {
	return guestAgentInterface.InterfaceName == ""
}

func updateVMIIfaceStatusWithGuestAgentData(ifaceStatus *v1.VirtualMachineInstanceNetworkInterface, guestAgentIface api.InterfaceStatus) {
	ifaceStatus.InterfaceName = guestAgentIface.InterfaceName
	ifaceStatus.IP = guestAgentIface.Ip
	ifaceStatus.IPs = guestAgentIface.IPs
}

func newVMIIfaceStatusFromGuestAgentData(guestAgentInterface api.InterfaceStatus) v1.VirtualMachineInstanceNetworkInterface {
	return v1.VirtualMachineInstanceNetworkInterface{
		MAC:           guestAgentInterface.Mac,
		IP:            guestAgentInterface.Ip,
		IPs:           guestAgentInterface.IPs,
		InterfaceName: guestAgentInterface.InterfaceName,
	}
}

func filterHostDevicesByAlias(hostDevices []api.HostDevice, prefix string) []api.HostDevice {
	var filteredHostDevices []api.HostDevice

	for _, hostDevice := range hostDevices {
		if hostDevice.Alias != nil && strings.HasPrefix(hostDevice.Alias.GetName(), prefix) {
			filteredHostDevices = append(filteredHostDevices, hostDevice)
		}
	}
	return filteredHostDevices
}

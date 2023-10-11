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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package netpod

import (
	"errors"
	"fmt"
	"os"

	knet "k8s.io/utils/net"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/network/driver/nmstate"
)

// discover goes over the current pod network configuration and persists/caches
// the relevant data (for recovery and data sharing).
func (n NetPod) discover(currentStatus *nmstate.Status) error {
	podIfaceStatusByName := ifaceStatusByName(currentStatus.Interfaces)
	podIfaceNameByVMINetwork := createNetworkNameScheme(n.vmiSpecNets, currentStatus.Interfaces)

	for _, vmiSpecIface := range n.vmiSpecIfaces {
		podIfaceName := podIfaceNameByVMINetwork[vmiSpecIface.Name]

		// The discovery is not relevant for interfaces marked for removal.
		if vmiSpecIface.State == v1.InterfaceStateAbsent {
			continue
		}

		podIfaceStatus, podIfaceExists := podIfaceStatusByName[podIfaceName]

		switch {
		case vmiSpecIface.Bridge != nil:
			if !podIfaceExists {
				return fmt.Errorf("pod link (%s) is missing", podIfaceName)
			}

			if err := n.storePodInterfaceData(vmiSpecIface, podIfaceStatus); err != nil {
				return err
			}

			if err := n.storeBridgeBindingDHCPInterfaceData(currentStatus, podIfaceStatus, vmiSpecIface, podIfaceName); err != nil {
				return err
			}

			if err := n.storeBridgeDomainInterfaceData(podIfaceStatus, vmiSpecIface); err != nil {
				return err
			}

		case vmiSpecIface.Masquerade != nil:
			if !podIfaceExists {
				return fmt.Errorf("pod link (%s) is missing", podIfaceName)
			}

			if err := n.storePodInterfaceData(vmiSpecIface, podIfaceStatus); err != nil {
				return err
			}

		case vmiSpecIface.Passt != nil:
			if !podIfaceExists {
				return fmt.Errorf("pod link (%s) is missing", podIfaceName)
			}

			if err := n.storePodInterfaceData(vmiSpecIface, podIfaceStatus); err != nil {
				return err
			}

		case vmiSpecIface.Slirp != nil:
			if !podIfaceExists {
				return fmt.Errorf("pod link (%s) is missing", podIfaceName)
			}

			if err := n.storePodInterfaceData(vmiSpecIface, podIfaceStatus); err != nil {
				return err
			}

		// Skip the discovery for all other known network interface bindings.
		case vmiSpecIface.Binding != nil:
		case vmiSpecIface.Macvtap != nil:
		case vmiSpecIface.SRIOV != nil:
		default:
			return fmt.Errorf("undefined binding method: %v", vmiSpecIface)
		}
	}
	return nil
}

func (n NetPod) storePodInterfaceData(vmiSpecIface v1.Interface, ifaceState nmstate.Interface) error {
	ifCache, err := cache.ReadPodInterfaceCache(n.cacheCreator, n.vmiUID, vmiSpecIface.Name)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to read pod interface cache for %s: %v", vmiSpecIface.Name, err)
		}
		ifCache = &cache.PodIfaceCacheData{}
	}

	ifCache.Iface = &vmiSpecIface

	ipv4 := firstIPGlobalUnicast(ifaceState.IPv4)
	ipv6 := firstIPGlobalUnicast(ifaceState.IPv6)
	switch {
	case ipv4 != nil && ipv6 != nil:
		ifCache.PodIPs, err = sortIPsBasedOnPrimaryIP(ipv4.IP, ipv6.IP)
		if err != nil {
			return err
		}
	case ipv4 != nil:
		ifCache.PodIPs = []string{ipv4.IP}
	case ipv6 != nil:
		ifCache.PodIPs = []string{ipv6.IP}
	default:
		return nil
	}
	ifCache.PodIP = ifCache.PodIPs[0]

	if err := cache.WritePodInterfaceCache(n.cacheCreator, n.vmiUID, vmiSpecIface.Name, ifCache); err != nil {
		log.Log.Reason(err).Errorf("failed to write pod interface data to cache")
		return err
	}
	return nil
}

// sortIPsBasedOnPrimaryIP returns a sorted slice of IP/s based on the detected cluster primary IP.
// The operation clones the Pod status IP list order logic.
func sortIPsBasedOnPrimaryIP(ipv4, ipv6 string) ([]string, error) {
	ipv4Primary, err := isIPv4Primary()
	if err != nil {
		return nil, err
	}

	if ipv4Primary {
		return []string{ipv4, ipv6}, nil
	}
	return []string{ipv6, ipv4}, nil
}

// isIPv4Primary inspects the existence of `MY_POD_IP` environment variable which
// is added to the virt-handler on its pod definition.
// The value tracks the pod `status.podIP` field value which in turn determines
// what is the cluster primary stack (IPv4 or IPv6).
func isIPv4Primary() (bool, error) {
	podIP, exist := os.LookupEnv("MY_POD_IP")
	if !exist {
		return false, fmt.Errorf("MY_POD_IP doesnt exists")
	}

	return !knet.IsIPv6String(podIP), nil
}

/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
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
	podIfaceNameByVMINetwork := createNetworkNameScheme(n.vmiSpecNets, n.vmiIfaceStatuses, currentStatus.Interfaces)

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

		case vmiSpecIface.Masquerade != nil:
			if !podIfaceExists {
				return fmt.Errorf("pod link (%s) is missing", podIfaceName)
			}

			if err := n.storePodInterfaceData(vmiSpecIface, podIfaceStatus); err != nil {
				return err
			}

		case vmiSpecIface.Binding != nil:
			// The existence of the pod interface depends on the network binding plugin implementation
			if podIfaceExists {
				if err := n.storePodInterfaceData(vmiSpecIface, podIfaceStatus); err != nil {
					return err
				}
			}
		case vmiSpecIface.PasstBinding != nil:
			if !podIfaceExists {
				return fmt.Errorf("pod link (%s) is missing", podIfaceName)
			}

			if err := n.storePodInterfaceData(vmiSpecIface, podIfaceStatus); err != nil {
				return err
			}

		// Skip the discovery for other known network interface bindings.
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

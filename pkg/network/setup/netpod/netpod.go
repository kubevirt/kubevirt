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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package netpod

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"

	"kubevirt.io/kubevirt/pkg/pointer"

	"kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/network/driver/nmstate"
	"kubevirt.io/kubevirt/pkg/network/driver/procsys"
	neterrors "kubevirt.io/kubevirt/pkg/network/errors"
	"kubevirt.io/kubevirt/pkg/network/link"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/network/netmachinery"
	"kubevirt.io/kubevirt/pkg/network/setup/netpod/masquerade"
	"kubevirt.io/kubevirt/pkg/network/vmispec"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	"kubevirt.io/client-go/log"

	v1 "kubevirt.io/api/core/v1"
)

type nmstateAdapter interface {
	Apply(spec *nmstate.Spec) error
	Read() (*nmstate.Status, error)
}

type masqueradeAdapter interface {
	Setup(bridgeIfaceSpec, podIfaceSpec *nmstate.Interface, vmiIface v1.Interface) error
}

type cacheCreator interface {
	New(filePath string) *cache.Cache
}

type NetPod struct {
	vmiSpecIfaces []v1.Interface
	vmiSpecNets   []v1.Network
	vmiUID        string
	podPID        int
	ownerID       int
	queuesCap     int

	nmstateAdapter    nmstateAdapter
	masqueradeAdapter masqueradeAdapter

	cacheCreator cacheCreator
}

type option func(*NetPod)

func NewNetPod(vmiNetworks []v1.Network, vmiIfaces []v1.Interface, vmiUID string, podPID, ownerID, queuesCapacity int, opts ...option) NetPod {
	n := NetPod{
		vmiSpecIfaces: vmiIfaces,
		vmiSpecNets:   vmiNetworks,
		vmiUID:        vmiUID,
		podPID:        podPID,
		ownerID:       ownerID,
		queuesCap:     queuesCapacity,

		nmstateAdapter:    nmstate.New(),
		masqueradeAdapter: masquerade.New(),

		cacheCreator: cache.CacheCreator{},
	}
	for _, opt := range opts {
		opt(&n)
	}
	return n
}

func WithNMStateAdapter(h nmstateAdapter) option {
	return func(n *NetPod) {
		n.nmstateAdapter = h
	}
}

func WithMasqueradeAdapter(h masqueradeAdapter) option {
	return func(n *NetPod) {
		n.masqueradeAdapter = h
	}
}

func WithCacheCreator(c cacheCreator) option {
	return func(n *NetPod) {
		n.cacheCreator = c
	}
}

func (n NetPod) Setup(postDiscoveryHook func() error) error {
	currentStatus, err := n.nmstateAdapter.Read()
	if err != nil {
		return err
	}

	currentStatusBytes, err := json.Marshal(currentStatus)
	if err != nil {
		return err
	}
	log.Log.Infof("Current pod network: %s", currentStatusBytes)

	if derr := n.discover(currentStatus); derr != nil {
		return derr
	}

	if err = postDiscoveryHook(); err != nil {
		return err
	}

	if err = n.config(currentStatus); err != nil {
		log.Log.Reason(err).Errorf("failed to configure pod network")
		return neterrors.CreateCriticalNetworkError(err)
	}
	return nil
}

func (n NetPod) config(currentStatus *nmstate.Status) error {
	desiredSpec, err := n.composeDesiredSpec(currentStatus)
	if err != nil {
		return err
	}

	desiredSpecBytes, err := json.Marshal(desiredSpec)
	if err != nil {
		return err
	}
	log.Log.Infof("Desired pod network: %s", desiredSpecBytes)

	if err = n.nmstateAdapter.Apply(desiredSpec); err != nil {
		return err
	}

	// Configuring NAT (nftables) is temporary done outside nmstate.
	// This should be eventually embedded into the nmstate desired state and applied by it.
	return n.setupNAT(desiredSpec, currentStatus)
}

func (n NetPod) composeDesiredSpec(currentStatus *nmstate.Status) (*nmstate.Spec, error) {
	podIfaceStatusByName := ifaceStatusByName(currentStatus.Interfaces)

	podIfaceNameByVMINetwork := createNetworkNameScheme(n.vmiSpecNets, currentStatus.Interfaces)

	spec := nmstate.Spec{Interfaces: []nmstate.Interface{}}

	for ifIndex, iface := range n.vmiSpecIfaces {
		var (
			ifacesSpec []nmstate.Interface
			err        error
		)
		podIfaceName := podIfaceNameByVMINetwork[iface.Name]

		// Filter out network interfaces marked for removal.
		// TODO: Support in the same flow the removal of such interfaces.
		if iface.State == v1.InterfaceStateAbsent {
			continue
		}

		switch {
		case iface.Bridge != nil:
			if _, exists := podIfaceStatusByName[podIfaceName]; !exists {
				return nil, fmt.Errorf("pod link (%s) is missing", podIfaceName)
			}
			ifacesSpec, err = n.bridgeBindingSpec(podIfaceName, ifIndex, podIfaceStatusByName)

			if nmstate.AnyInterface(ifacesSpec, hasIP4GlobalUnicast) {
				spec.LinuxStack.IPv4.ArpIgnore = pointer.P(procsys.ARPReplyMode1)
			}
		case iface.Masquerade != nil:
			if _, exists := podIfaceStatusByName[podIfaceName]; !exists {
				return nil, fmt.Errorf("pod link (%s) is missing", podIfaceName)
			}
			ifacesSpec, err = n.masqueradeBindingSpec(podIfaceName, ifIndex, podIfaceStatusByName)

			if nmstate.AnyInterface(ifacesSpec, hasIP4GlobalUnicast) {
				spec.LinuxStack.IPv4.Forwarding = pointer.P(true)
			}
			if nmstate.AnyInterface(ifacesSpec, hasIP6GlobalUnicast) {
				spec.LinuxStack.IPv6.Forwarding = pointer.P(true)
			}
		case iface.Passt != nil:
			spec.LinuxStack.IPv4.PingGroupRange = []int{107, 107}
			spec.LinuxStack.IPv4.UnprivilegedPortStart = pointer.P(0)
		case iface.Macvtap != nil:
		case iface.SRIOV != nil:
		case iface.Slirp != nil:
		case iface.Binding != nil:
		default:
			return nil, fmt.Errorf("undefined binding method: %v", iface)
		}
		if err != nil {
			return nil, err
		}
		spec.Interfaces = append(spec.Interfaces, ifacesSpec...)
	}

	return &spec, nil
}

func (n NetPod) bridgeBindingSpec(podIfaceName string, vmiIfaceIndex int, ifaceStatusByName map[string]nmstate.Interface) ([]nmstate.Interface, error) {
	const (
		bridgeFakeIPBase = "169.254.75.1"
		bridgeFakePrefix = 32
	)

	vmiNetworkName := n.vmiSpecIfaces[vmiIfaceIndex].Name

	bridgeIface := nmstate.Interface{
		Name:     link.GenerateBridgeName(podIfaceName),
		TypeName: nmstate.TypeBridge,
		State:    nmstate.IfaceStateUp,
		Ethtool:  nmstate.Ethtool{Feature: nmstate.Feature{TxChecksum: pointer.P(false)}},
		Metadata: &nmstate.IfaceMetadata{NetworkName: vmiNetworkName},
	}

	podStatusIface := ifaceStatusByName[podIfaceName]

	if hasIPGlobalUnicast(podStatusIface.IPv4) {
		bridgeIface.IPv4 = nmstate.IP{
			Enabled: pointer.P(true),
			Address: []nmstate.IPAddress{
				{
					IP:        bridgeFakeIPBase + strconv.Itoa(vmiIfaceIndex),
					PrefixLen: bridgeFakePrefix,
				},
			},
		}
	}

	podIfaceAlternativeName := link.GenerateNewBridgedVmiInterfaceName(podIfaceName)
	podIface := nmstate.Interface{
		Index:       podStatusIface.Index,
		Name:        podIfaceAlternativeName,
		CopyMacFrom: bridgeIface.Name,
		Controller:  bridgeIface.Name,
		IPv4:        nmstate.IP{Enabled: pointer.P(false)},
		IPv6:        nmstate.IP{Enabled: pointer.P(false)},
		LinuxStack:  nmstate.LinuxIfaceStack{PortLearning: pointer.P(false)},
		Metadata:    &nmstate.IfaceMetadata{NetworkName: vmiNetworkName},
	}

	tapIface := nmstate.Interface{
		Name:       link.GenerateTapDeviceName(podIfaceName),
		TypeName:   nmstate.TypeTap,
		State:      nmstate.IfaceStateUp,
		MTU:        podStatusIface.MTU,
		Controller: bridgeIface.Name,
		Tap: &nmstate.TapDevice{
			Queues: n.networkQueues(vmiIfaceIndex),
			UID:    n.ownerID,
			GID:    n.ownerID,
		},
		Metadata: &nmstate.IfaceMetadata{Pid: n.podPID, NetworkName: vmiNetworkName},
	}

	dummyIface := nmstate.Interface{
		Name:     podIfaceName,
		TypeName: nmstate.TypeDummy,
		MTU:      podStatusIface.MTU,
		IPv4:     podStatusIface.IPv4,
		IPv6:     podStatusIface.IPv6,
		Metadata: &nmstate.IfaceMetadata{NetworkName: vmiNetworkName},
	}

	return []nmstate.Interface{bridgeIface, podIface, tapIface, dummyIface}, nil
}

func (n NetPod) networkQueues(vmiIfaceIndex int) int {
	ifaceModel := n.vmiSpecIfaces[vmiIfaceIndex].Model
	if ifaceModel == "" {
		ifaceModel = v1.VirtIO
	}
	var queues int
	if ifaceModel == v1.VirtIO {
		queues = n.queuesCap
	}
	return queues
}

func (n NetPod) masqueradeBindingSpec(podIfaceName string, vmiIfaceIndex int, ifaceStatusByName map[string]nmstate.Interface) ([]nmstate.Interface, error) {
	podIface := ifaceStatusByName[podIfaceName]

	vmiNetworkName := n.vmiSpecIfaces[vmiIfaceIndex].Name
	vmiNetwork := vmispec.LookupNetworkByName(n.vmiSpecNets, vmiNetworkName)

	bridgeIface := nmstate.Interface{
		Name:       link.GenerateBridgeName(podIfaceName),
		TypeName:   nmstate.TypeBridge,
		State:      nmstate.IfaceStateUp,
		MacAddress: link.StaticMasqueradeBridgeMAC,
		MTU:        podIface.MTU,
		Ethtool:    nmstate.Ethtool{Feature: nmstate.Feature{TxChecksum: pointer.P(false)}},
		IPv4:       nmstate.IP{Enabled: pointer.P(false)},
		IPv6:       nmstate.IP{Enabled: pointer.P(false)},
		Metadata:   &nmstate.IfaceMetadata{NetworkName: vmiNetwork.Name},
	}

	if hasIPGlobalUnicast(podIface.IPv4) {
		ip4GatewayAddress, err := gatewayIP(vmiNetwork.Pod.VMNetworkCIDR, api.DefaultVMCIDR)
		if err != nil {
			return nil, err
		}
		bridgeIface.IPv4 = nmstate.IP{
			Enabled: pointer.P(true),
			Address: []nmstate.IPAddress{ip4GatewayAddress},
		}
		bridgeIface.LinuxStack.IP4RouteLocalNet = pointer.P(true)
	}

	if hasIPGlobalUnicast(podIface.IPv6) {
		ip6GatewayAddress, err := gatewayIP(vmiNetwork.Pod.VMIPv6NetworkCIDR, api.DefaultVMIpv6CIDR)
		if err != nil {
			return nil, err
		}
		bridgeIface.IPv6 = nmstate.IP{
			Enabled: pointer.P(true),
			Address: []nmstate.IPAddress{ip6GatewayAddress},
		}
	}

	tapIface := nmstate.Interface{
		Name:       link.GenerateTapDeviceName(podIfaceName),
		TypeName:   nmstate.TypeTap,
		State:      nmstate.IfaceStateUp,
		MTU:        podIface.MTU,
		Controller: bridgeIface.Name,
		Tap: &nmstate.TapDevice{
			Queues: n.networkQueues(vmiIfaceIndex),
			UID:    n.ownerID,
			GID:    n.ownerID,
		},
		Metadata: &nmstate.IfaceMetadata{Pid: n.podPID, NetworkName: vmiNetwork.Name},
	}

	return []nmstate.Interface{bridgeIface, tapIface}, nil
}

func (n NetPod) setupNAT(desiredSpec *nmstate.Spec, currentStatus *nmstate.Status) error {
	bridgeIfaceSpec := n.lookupMasquradeBridge(desiredSpec.Interfaces)
	if bridgeIfaceSpec == nil {
		return nil
	}
	podIfaceNameByVMINetwork := createNetworkNameScheme(n.vmiSpecNets, currentStatus.Interfaces)
	podIfaceName := podIfaceNameByVMINetwork[bridgeIfaceSpec.Metadata.NetworkName]
	podIfaceSpec := nmstate.LookupInterface(currentStatus.Interfaces, func(i nmstate.Interface) bool {
		return i.Name == podIfaceName
	})
	if podIfaceSpec == nil {
		return fmt.Errorf("setup-nat: pod link (%s) is missing", podIfaceName)
	}
	vmiIface := vmispec.FilterInterfacesSpec(n.vmiSpecIfaces, func(i v1.Interface) bool {
		return i.Name == bridgeIfaceSpec.Metadata.NetworkName
	})
	return n.masqueradeAdapter.Setup(bridgeIfaceSpec, podIfaceSpec, vmiIface[0])
}

func (n NetPod) lookupMasquradeBridge(desiredIfacesSpec []nmstate.Interface) *nmstate.Interface {
	masqueradeIfaces := vmispec.FilterInterfacesSpec(n.vmiSpecIfaces, func(i v1.Interface) bool {
		return i.Masquerade != nil
	})
	if len(masqueradeIfaces) > 0 {
		vmiMasqIface := masqueradeIfaces[0]
		bridgeIfaceSpec := nmstate.LookupInterface(desiredIfacesSpec, func(i nmstate.Interface) bool {
			return i.Metadata != nil && i.Metadata.NetworkName == vmiMasqIface.Name && i.TypeName == nmstate.TypeBridge
		})

		return bridgeIfaceSpec
	}
	return nil
}

func ifaceStatusByName(interfaces []nmstate.Interface) map[string]nmstate.Interface {
	ifaceByName := map[string]nmstate.Interface{}
	for _, iface := range interfaces {
		ifaceByName[iface.Name] = iface
	}
	return ifaceByName
}

func gatewayIP(cidr, defaultCIDR string) (nmstate.IPAddress, error) {
	if cidr == "" {
		cidr = defaultCIDR
	}
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nmstate.IPAddress{}, fmt.Errorf("failed to parse VM CIDR: %s, %v", cidr, err)
	}
	const minMaskBitsForHostAddresses = 2
	if prefixLen, maxPrefixLen := ipNet.Mask.Size(); prefixLen > maxPrefixLen-minMaskBitsForHostAddresses {
		return nmstate.IPAddress{}, fmt.Errorf("VM CIDR subnet is too small, at least 2 host addresses are required: %s", cidr)
	}
	netmachinery.NextIP(ipNet.IP)

	gatewayAddress := ipNet.IP.String()
	ipGatewayPrefixLen, _ := ipNet.Mask.Size()

	return nmstate.IPAddress{
		IP:        gatewayAddress,
		PrefixLen: ipGatewayPrefixLen,
	}, nil
}

func hasIP4GlobalUnicast(iface nmstate.Interface) bool {
	return hasIPGlobalUnicast(iface.IPv4)
}

func hasIP6GlobalUnicast(iface nmstate.Interface) bool {
	return hasIPGlobalUnicast(iface.IPv4)
}

func hasIPGlobalUnicast(ip nmstate.IP) bool {
	return firstIPGlobalUnicast(ip) != nil
}

func firstIPGlobalUnicast(ip nmstate.IP) *nmstate.IPAddress {
	if ip.Enabled != nil && *ip.Enabled {
		for _, addr := range ip.Address {
			if net.ParseIP(addr.IP).IsGlobalUnicast() {
				address := addr
				return &address
			}
		}
	}
	return nil
}

func createNetworkNameScheme(networks []v1.Network, currentIfaces []nmstate.Interface) map[string]string {
	if includesOrdinalNames(currentIfaces) {
		return namescheme.CreateOrdinalNetworkNameScheme(networks)
	}
	return namescheme.CreateHashedNetworkNameScheme(networks)
}

func includesOrdinalNames(ifaces []nmstate.Interface) bool {
	for _, iface := range ifaces {
		if namescheme.OrdinalSecondaryInterfaceName(iface.Name) {
			return true
		}
	}
	return false
}

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

	k8serrors "k8s.io/apimachinery/pkg/util/errors"

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

type NSExecutor interface {
	Do(func() error) error
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
	state        *State
}

type option func(*NetPod)

func NewNetPod(vmiNetworks []v1.Network, vmiIfaces []v1.Interface, vmiUID string, podPID, ownerID, queuesCapacity int, state *State, opts ...option) NetPod {
	n := NetPod{
		vmiSpecIfaces: vmiIfaces,
		vmiSpecNets:   vmiNetworks,
		vmiUID:        vmiUID,
		podPID:        podPID,
		ownerID:       ownerID,
		queuesCap:     queuesCapacity,
		state:         state,

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

func (n NetPod) Setup() error {
	// Not all network bindings are processed in the network setup.
	filteredNets, err := filterSupportedBindingNetworks(n.vmiSpecNets, n.vmiSpecIfaces)
	if err != nil {
		return err
	}

	pendingNets, startedNets, finishedNets, err := n.state.PendingStartedFinished(filteredNets)
	if err != nil {
		return err
	}
	if err := n.validateNoNetworkReconfigured(startedNets); err != nil {
		return err
	}

	unplugIfaces := n.unplugInterfaces(startedNets, finishedNets)

	// The pending networks should not include networks that are marked for removal.
	// Filter out such networks for the pending network list.
	pendingNets = vmispec.FilterNetworksSpec(pendingNets, func(net v1.Network) bool {
		iface := vmispec.LookupInterfaceByName(n.vmiSpecIfaces, net.Name)
		return iface != nil && iface.State != v1.InterfaceStateAbsent
	})
	if len(pendingNets) == 0 && len(unplugIfaces) == 0 {
		return nil
	}

	err = n.state.NSExec.Do(func() error {
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

		if serr := n.state.SetStarted(pendingNets); serr != nil {
			return serr
		}

		if err = n.config(currentStatus); err != nil {
			log.Log.Reason(err).Errorf("failed to configure pod network")
			return neterrors.CreateCriticalNetworkError(err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	if serr := n.state.SetFinished(pendingNets); serr != nil {
		return serr
	}

	unplugNetworks := vmispec.FilterNetworksByInterfaces(n.vmiSpecNets, unplugIfaces)
	if serr := n.clearCache(unplugNetworks); serr != nil {
		return serr
	}

	return nil
}

func (n NetPod) validateNoNetworkReconfigured(startedNets []v1.Network) error {
	if len(startedNets) > 0 {
		for _, net := range startedNets {
			startedIface := vmispec.LookupInterfaceByName(n.vmiSpecIfaces, net.Name)
			if startedIface != nil && startedIface.State != v1.InterfaceStateAbsent {
				return neterrors.CreateCriticalNetworkError(
					fmt.Errorf("preparation for networks %v cannot be restarted", startedNets),
				)
			}
		}
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

		switch {
		case iface.Bridge != nil:
			// A missing pod interface is not considered an error in case the interface is marked for removal.
			if _, exists := podIfaceStatusByName[podIfaceName]; !exists && iface.State != v1.InterfaceStateAbsent {
				return nil, fmt.Errorf("pod link (%s) is missing", podIfaceName)
			}
			ifacesSpec, err = n.bridgeBindingSpec(podIfaceName, ifIndex, podIfaceStatusByName)

			if nmstate.AnyInterface(ifacesSpec, hasIP4GlobalUnicast) {
				spec.LinuxStack.IPv4.ArpIgnore = pointer.P(procsys.ARPReplyMode1)
			}

			if iface.State == v1.InterfaceStateAbsent {
				var filteredIfacesSpec []nmstate.Interface
				for _, ifaceSpec := range ifacesSpec {
					// Interfaces with no type are not owned by kubevirt, therefore not removed.
					if ifaceSpec.TypeName != "" {
						ifaceSpec.State = nmstate.IfaceStateAbsent
						filteredIfacesSpec = append(filteredIfacesSpec, ifaceSpec)
					}
				}
				ifacesSpec = filteredIfacesSpec
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

	podIfaceAlternativeName := link.GenerateNewBridgedVmiInterfaceName(podIfaceName)
	podStatusIface, exist := ifaceStatusByName[podIfaceAlternativeName]
	if !exist {
		podStatusIface = ifaceStatusByName[podIfaceName]
	}

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

func filterSupportedBindingNetworks(specNetworks []v1.Network, specInterfaces []v1.Interface) ([]v1.Network, error) {
	var networks []v1.Network
	for _, network := range specNetworks {
		iface := vmispec.LookupInterfaceByName(specInterfaces, network.Name)
		if iface == nil {
			return nil, fmt.Errorf("no iface matching with network %s", network.Name)
		}

		if iface.Binding != nil || iface.SRIOV != nil || iface.Macvtap != nil {
			continue
		}

		networks = append(networks, network)
	}

	return networks, nil
}

func (n NetPod) unplugInterfaces(startedNets, finishedNets []v1.Network) []v1.Interface {
	nonPendingNetworks := append(startedNets, finishedNets...)
	nonPendingNetsByName := vmispec.IndexNetworkSpecByName(nonPendingNetworks)
	unplugIfaces := vmispec.FilterInterfacesSpec(n.vmiSpecIfaces, func(iface v1.Interface) bool {
		_, netExists := nonPendingNetsByName[iface.Name]
		return iface.State == v1.InterfaceStateAbsent && netExists
	})
	return unplugIfaces
}

func (n NetPod) clearCache(nets []v1.Network) error {
	var unplugErrors []error
	for _, net := range nets {
		err := cache.DeleteDomainInterfaceCache(n.cacheCreator, strconv.Itoa(n.podPID), net.Name)
		if err != nil {
			unplugErrors = append(unplugErrors, err)
		}

		podInterfaceName := namescheme.HashedPodInterfaceName(net)
		err = cache.DeleteDHCPInterfaceCache(n.cacheCreator, strconv.Itoa(n.podPID), podInterfaceName)
		if err != nil {
			unplugErrors = append(unplugErrors, err)
		}

		// the PodInterface cache should be the last one to be cleaned.
		// It should be cleaned as the last step of the cleanup, since it is the indicator the cleanup should be done/not over yet.
		if len(unplugErrors) == 0 {
			err = cache.DeletePodInterfaceCache(n.cacheCreator, n.vmiUID, net.Name)
			if err != nil {
				unplugErrors = append(unplugErrors, err)
			}
		}
	}

	if len(unplugErrors) > 0 {
		return k8serrors.NewAggregate(unplugErrors)
	}
	return n.state.Delete(nets)
}

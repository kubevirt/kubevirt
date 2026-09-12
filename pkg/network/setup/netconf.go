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
 * Copyright The KubeVirt Authors.
 *
 */

package network

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"kubevirt.io/client-go/log"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/network/istio"
	"kubevirt.io/kubevirt/pkg/network/netns"
	"kubevirt.io/kubevirt/pkg/network/setup/netpod"
	"kubevirt.io/kubevirt/pkg/network/setup/netpod/masquerade"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/util"
	converternet "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/network"
	"kubevirt.io/kubevirt/pkg/vmitrait"
)

type cacheCreator interface {
	New(filePath string) *cache.Cache
}

type clusterConfigurer interface {
	GetNetworkBindings() map[string]v1.InterfaceBindingPlugin
	PortRangesSpecGateEnabled() bool
}

type NetConf struct {
	cacheCreator     cacheCreator
	nsFactory        nsFactory
	state            map[string]*netpod.State
	configStateMutex *sync.RWMutex

	clusterConfigurer clusterConfigurer
}

type nsFactory func(int) NSExecutor

type NSExecutor interface {
	Do(func() error) error
}

func NewNetConf(clusterConfigurer clusterConfigurer) *NetConf {
	var cacheFactory cache.CacheCreator
	return NewNetConfWithCustomFactoryAndConfigState(func(pid int) NSExecutor {
		return netns.New(pid)
	}, cacheFactory, map[string]*netpod.State{}, clusterConfigurer)
}

func NewNetConfWithCustomFactoryAndConfigState(nsFactory nsFactory, cacheCreator cacheCreator, state map[string]*netpod.State, clusterConfigurer clusterConfigurer) *NetConf {
	return &NetConf{
		state:             state,
		configStateMutex:  &sync.RWMutex{},
		cacheCreator:      cacheCreator,
		nsFactory:         nsFactory,
		clusterConfigurer: clusterConfigurer,
	}
}

// networkConfigStateKey returns a key for tracking network preparation state.
// The key includes the active virt-launcher pod UID so a new pod always starts
// with a fresh network setup, even when the VMI UID is unchanged.
func networkConfigStateKey(vmi *v1.VirtualMachineInstance) string {
	vmiUID := string(vmi.UID)
	nodeName := vmi.Status.NodeName
	if nodeName == "" || len(vmi.Status.ActivePods) == 0 {
		return vmiUID
	}

	var matchingPodUIDs []string
	for podUID, podNodeName := range vmi.Status.ActivePods {
		if podNodeName == nodeName {
			matchingPodUIDs = append(matchingPodUIDs, string(podUID))
		}
	}

	switch len(matchingPodUIDs) {
	case 0:
		return vmiUID
	case 1:
		return vmiUID + "/" + matchingPodUIDs[0]
	default:
		sort.Strings(matchingPodUIDs)
		log.Log.Object(vmi).Warningf(
			"multiple active pods (%v) found on node %s; using %s for network state key",
			matchingPodUIDs, nodeName, matchingPodUIDs[0],
		)
		return vmiUID + "/" + matchingPodUIDs[0]
	}
}

func deleteNetworkConfigStatesForVMI(state map[string]*netpod.State, vmiUID string) {
	for key := range state {
		if key == vmiUID || strings.HasPrefix(key, vmiUID+"/") {
			delete(state, key)
		}
	}
}

// Setup applies (privilege) network related changes for an existing virt-launcher pod.
func (c *NetConf) Setup(vmi *v1.VirtualMachineInstance, networks []v1.Network, launcherPid int) error {
	stateKey := networkConfigStateKey(vmi)
	c.configStateMutex.RLock()
	state, ok := c.state[stateKey]
	c.configStateMutex.RUnlock()
	if !ok {
		configStateCache := NewConfigStateCache(string(vmi.UID), c.cacheCreator)
		ns := c.nsFactory(launcherPid)
		state = netpod.NewState(&configStateCache, ns)
		c.configStateMutex.Lock()
		c.state[stateKey] = state
		c.configStateMutex.Unlock()
	}

	ownerID, _ := strconv.Atoi(netdriver.LibvirtUserAndGroupId)
	if vmitrait.IsNonRoot(vmi) {
		ownerID = util.NonRootUID
	}
	queuesCapacity := int(converternet.NetworkQueuesCapacity(vmi))
	netpod := netpod.NewNetPod(
		networks,
		vmispec.FilterInterfacesByNetworks(vmi.Spec.Domain.Devices.Interfaces, networks),
		string(vmi.UID),
		launcherPid,
		ownerID,
		queuesCapacity,
		state,
		netpod.WithMasqueradeAdapter(newMasqueradeAdapter(vmi, c.clusterConfigurer.PortRangesSpecGateEnabled())),
		netpod.WithCacheCreator(c.cacheCreator),
		netpod.WithBindingPlugins(c.clusterConfigurer.GetNetworkBindings()),
		netpod.WithLogger(log.Log.Object(vmi)),
		netpod.WithVMIIfaceStatuses(vmi.Status.Interfaces),
	)

	if err := netpod.Setup(); err != nil {
		return fmt.Errorf("setup failed, err: %w", err)
	}
	return nil
}

func (c *NetConf) Teardown(vmi *v1.VirtualMachineInstance) error {
	c.configStateMutex.Lock()
	deleteNetworkConfigStatesForVMI(c.state, string(vmi.UID))
	c.configStateMutex.Unlock()
	podCache := cache.NewPodInterfaceCache(c.cacheCreator, string(vmi.UID))
	if err := podCache.Remove(); err != nil {
		return fmt.Errorf("teardown failed, err: %w", err)
	}

	return nil
}

func newMasqueradeAdapter(vmi *v1.VirtualMachineInstance, portRangesSpecGateEnabled bool) masquerade.MasqPod {
	if vmi.Status.MigrationTransport == v1.MigrationTransportUnix {
		return masquerade.New(
			masquerade.WithIstio(istio.ProxyInjectionEnabled(vmi)),
			masquerade.WithPortRangesSpecGateEnabled(portRangesSpecGateEnabled),
		)
	} else {
		return masquerade.New(
			masquerade.WithIstio(istio.ProxyInjectionEnabled(vmi)),
			masquerade.WithPortRangesSpecGateEnabled(portRangesSpecGateEnabled),
			masquerade.WithLegacyMigrationPorts(),
		)
	}
}

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
	"sync"

	netvmispec "kubevirt.io/kubevirt/pkg/network/vmispec"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/namescheme"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/network/netns"
)

type cacheCreator interface {
	New(filePath string) *cache.Cache
}

type ConfigStateExecutor interface {
	Unplug(networks []v1.Network, filterFunc func([]v1.Network) ([]string, error), cleanupFunc func(string) error) error
	Run(nics []podNIC, preRunFunc func([]podNIC) ([]podNIC, error), discoverFunc func(*podNIC) error, configFunc func(*podNIC) error) error
}

type NetConf struct {
	cacheCreator     cacheCreator
	nsFactory        nsFactory
	configState      map[string]ConfigStateExecutor
	configStateMutex *sync.RWMutex
}

type nsFactory func(int) NSExecutor

type NSExecutor interface {
	Do(func() error) error
}

func NewNetConf() *NetConf {
	var cacheFactory cache.CacheCreator
	return NewNetConfWithCustomFactoryAndConfigState(func(pid int) NSExecutor {
		return netns.New(pid)
	}, cacheFactory, map[string]ConfigStateExecutor{})
}

func NewNetConfWithCustomFactoryAndConfigState(nsFactory nsFactory, cacheCreator cacheCreator, configState map[string]ConfigStateExecutor) *NetConf {
	return &NetConf{
		configState:      configState,
		configStateMutex: &sync.RWMutex{},
		cacheCreator:     cacheCreator,
		nsFactory:        nsFactory,
	}
}

// Setup applies (privilege) network related changes for an existing virt-launcher pod.
func (c *NetConf) Setup(vmi *v1.VirtualMachineInstance, networks []v1.Network, launcherPid int, preSetup func() error) error {
	if err := preSetup(); err != nil {
		return fmt.Errorf("setup failed at pre-setup stage, err: %w", err)
	}

	netConfigurator := NewVMNetworkConfigurator(vmi, c.cacheCreator, &launcherPid)

	c.configStateMutex.RLock()
	configState, ok := c.configState[string(vmi.UID)]
	c.configStateMutex.RUnlock()
	if !ok {
		cache := NewConfigStateCache(string(vmi.UID), c.cacheCreator)
		configStateCache, err := upgradeConfigStateCache(&cache, networks, c.cacheCreator, string(vmi.UID))
		if err != nil {
			return err
		}
		ns := c.nsFactory(launcherPid)
		newConfigState := NewConfigState(configStateCache, ns)
		configState = &newConfigState
		c.configStateMutex.Lock()
		c.configState[string(vmi.UID)] = configState
		c.configStateMutex.Unlock()
	}

	// Absent networks are passed as well since, Absent network with ordinary name has to be plugged
	err := netConfigurator.SetupPodNetworkPhase1(launcherPid, networks, configState)

	if err != nil {
		return fmt.Errorf("setup failed, err: %w", err)
	}

	absentIfaces := netvmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		return iface.State == v1.InterfaceStateAbsent
	})
	absentNets := netvmispec.FilterNetworksByInterfaces(vmi.Spec.Networks, absentIfaces)
	if len(absentIfaces) != 0 {
		err = c.hotUnplugInterfaces(vmi, absentNets, configState, launcherPid)
	}

	if err != nil {
		return err
	}
	return nil
}

func upgradeConfigStateCache(stateCache *ConfigStateCache, networks []v1.Network, cacheCreator cacheCreator, vmiUID string) (*ConfigStateCache, error) {
	for networkName, podIfaceName := range namescheme.CreateOrdinalNetworkNameScheme(networks) {
		exists, err := stateCache.Exists(podIfaceName)
		if err != nil {
			return nil, err
		}
		if exists {
			data, rErr := stateCache.Read(podIfaceName)
			if rErr != nil {
				return nil, rErr
			}
			if wErr := stateCache.Write(networkName, data); wErr != nil {
				return nil, wErr
			}
			if dErr := stateCache.Delete(podIfaceName); dErr != nil {
				log.Log.Reason(dErr).Errorf("failed to delete pod interface (%s) state from cache", podIfaceName)
			}
			if dErr := cache.DeletePodInterfaceCache(cacheCreator, vmiUID, podIfaceName); dErr != nil {
				log.Log.Reason(dErr).Errorf("failed to delete pod interface (%s) from cache", podIfaceName)
			}
		}
	}
	return stateCache, nil
}

func (c *NetConf) Teardown(vmi *v1.VirtualMachineInstance) error {
	c.configStateMutex.Lock()
	delete(c.configState, string(vmi.UID))
	c.configStateMutex.Unlock()
	podCache := cache.NewPodInterfaceCache(c.cacheCreator, string(vmi.UID))
	if err := podCache.Remove(); err != nil {
		return fmt.Errorf("teardown failed, err: %w", err)
	}

	return nil
}

func (c *NetConf) hotUnplugInterfaces(vmi *v1.VirtualMachineInstance, networks []v1.Network, configState ConfigStateExecutor, launcherPid int) error {
	netConfigurator := NewVMNetworkConfigurator(vmi, c.cacheCreator, &launcherPid)
	return netConfigurator.UnplugPodNetworksPhase1(vmi, networks, configState)
}

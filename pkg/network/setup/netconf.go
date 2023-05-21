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

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/namescheme"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/network/netns"
)

type cacheCreator interface {
	New(filePath string) *cache.Cache
}

type NetConf struct {
	cacheCreator     cacheCreator
	nsFactory        nsFactory
	configState      map[string]ConfigState
	configStateMutex *sync.RWMutex
}

type nsFactory func(int) NSExecutor

type NSExecutor interface {
	Do(func() error) error
}

func NewNetConf() *NetConf {
	var cacheFactory cache.CacheCreator
	return NewNetConfWithCustomFactory(func(pid int) NSExecutor {
		return netns.New(pid)
	}, cacheFactory)
}

func NewNetConfWithCustomFactory(nsFactory nsFactory, cacheCreator cacheCreator) *NetConf {
	return &NetConf{
		configState:      map[string]ConfigState{},
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

	netConfigurator := NewVMNetworkConfigurator(vmi, c.cacheCreator)

	c.configStateMutex.RLock()
	configState, ok := c.configState[string(vmi.UID)]
	c.configStateMutex.RUnlock()
	if !ok {
		cache := NewConfigStateCache(string(vmi.UID), c.cacheCreator)
		configStateCache, err := upgradeConfigStateCache(&cache, networks)
		if err != nil {
			return err
		}
		ns := c.nsFactory(launcherPid)
		configState = NewConfigState(configStateCache, ns)
		c.configStateMutex.Lock()
		c.configState[string(vmi.UID)] = configState
		c.configStateMutex.Unlock()
	}

	err := netConfigurator.SetupPodNetworkPhase1(launcherPid, networks, configState)
	if err != nil {
		return fmt.Errorf("setup failed, err: %w", err)
	}
	return nil
}

func upgradeConfigStateCache(stateCache *ConfigStateCache, networks []v1.Network) (*ConfigStateCache, error) {
	for networkName, podIfaceName := range namescheme.CreateNetworkNameScheme(networks) {
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
				log.Log.Reason(dErr).Errorf("failed to delete pod interface network (%s) state from cache", podIfaceName)
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

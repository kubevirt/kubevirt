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

	"kubevirt.io/kubevirt/pkg/network/netns"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/cache"
)

type cacheCreator interface {
	New(filePath string) *cache.Cache
}

type NetConf struct {
	setupCompleted sync.Map
	cacheCreator   cacheCreator
	nsFactory      nsFactory
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
		setupCompleted: sync.Map{},
		cacheCreator:   cacheCreator,
		nsFactory:      nsFactory,
	}
}

// Setup applies (privilege) network related changes for an existing virt-launcher pod.
// As the changes are performed in the virt-launcher network namespace, which is relative expensive,
// an early cache check is performed to avoid executing the same operation again (if the last one completed).
func (c *NetConf) Setup(vmi *v1.VirtualMachineInstance, launcherPid int, preSetup func() error) error {
	id := vmi.UID
	if _, exists := c.setupCompleted.Load(id); exists {
		return nil
	}

	if err := preSetup(); err != nil {
		return fmt.Errorf("setup failed at pre-setup stage, err: %w", err)
	}

	netConfigurator := NewVMNetworkConfigurator(vmi, c.cacheCreator)

	ns := c.nsFactory(launcherPid)
	err := ns.Do(func() error {
		return netConfigurator.SetupPodNetworkPhase1(launcherPid)
	})
	if err != nil {
		return fmt.Errorf("setup failed, err: %w", err)
	}

	c.setupCompleted.Store(id, struct{}{})

	return nil
}

func (c *NetConf) Teardown(vmi *v1.VirtualMachineInstance) error {
	c.setupCompleted.Delete(vmi.UID)
	podCache := cache.NewPodInterfaceCache(c.cacheCreator, string(vmi.UID))
	if err := podCache.Remove(); err != nil {
		return fmt.Errorf("teardown failed, err: %w", err)
	}

	return nil
}

// SetupCompleted examines if the setup on a given VMI completed.
// It uses the (soft) cache to determine the information.
func (c *NetConf) SetupCompleted(vmi *v1.VirtualMachineInstance) bool {
	_, exists := c.setupCompleted.Load(vmi.UID)
	return exists
}

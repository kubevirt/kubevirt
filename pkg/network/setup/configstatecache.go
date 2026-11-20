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
	"errors"
	"fmt"
	"os"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/cache"
)

const defaultState = cache.PodIfaceNetworkPreparationPending

type ConfigStateCache struct {
	vmiUID                string
	launcherPid           int
	cacheCreator          cacheCreator
	volatilePodIfaceState map[string]cache.PodIfaceState
}

func NewConfigStateCache(vmiUID string, launcherPid int, cacheCreator cacheCreator) ConfigStateCache {
	return ConfigStateCache{
		vmiUID:                vmiUID,
		launcherPid:           launcherPid,
		cacheCreator:          cacheCreator,
		volatilePodIfaceState: map[string]cache.PodIfaceState{},
	}
}

func NewConfigStateCacheWithPodIfaceStateData(vmiUID string, launcherPid int, cacheCreator cacheCreator, volatilePodIfaceState map[string]cache.PodIfaceState) ConfigStateCache {
	return ConfigStateCache{
		vmiUID:                vmiUID,
		launcherPid:           launcherPid,
		cacheCreator:          cacheCreator,
		volatilePodIfaceState: volatilePodIfaceState,
	}
}

func (c *ConfigStateCache) Read(key string) (cache.PodIfaceState, error) {
	if v, ok := c.volatilePodIfaceState[key]; ok {
		return v, nil
	}
	data, err := c.readPodIface(key)
	if err != nil {
		return defaultState, fmt.Errorf("failed to read pod interface network state from cache: %v", err)
	}
	state := defaultState
	if data != nil {
		state = data.State
	}
	c.volatilePodIfaceState[key] = state
	return state, nil
}

func (c *ConfigStateCache) Write(key string, state cache.PodIfaceState) error {
	podIfaceCacheData, err := c.readPodIface(key)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to read pod interface network (%s) state from cache", key)
		return err
	}
	if podIfaceCacheData == nil {
		podIfaceCacheData = &cache.PodIfaceCacheData{}
	}
	podIfaceCacheData.State = state
	if err := c.writePodIface(key, podIfaceCacheData); err != nil {
		log.Log.Reason(err).Errorf("failed to write pod interface network (%s) state to cache", key)
		return err
	}
	c.volatilePodIfaceState[key] = state
	return nil
}

func (c *ConfigStateCache) Delete(key string) error {
	delete(c.volatilePodIfaceState, key)
	podIfaceCacheData, err := c.readPodIface(key)
	if err != nil {
		return err
	}
	if podIfaceCacheData == nil {
		return nil
	}
	podIfaceCacheData.State = cache.PodIfaceNetworkPreparationPending
	return c.writePodIface(key, podIfaceCacheData)
}

// readPodIface reads PodIfaceCacheData from disk. Returns (nil, nil) when absent.
// Tries the launcher procfs path first, then the pre-upgrade virt-handler-local path.
func (c *ConfigStateCache) readPodIface(key string) (*cache.PodIfaceCacheData, error) {
	entry, err := cache.NewPodInterfaceCache(c.cacheCreator, c.launcherPid).IfaceEntry(key)
	if err == nil {
		data, err := entry.Read()
		if err == nil {
			return data, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	// Upgrade fallback: check pre-upgrade virt-handler-local path (read-only).
	return c.readPreUpgradeState(key)
}

func (c *ConfigStateCache) writePodIface(key string, data *cache.PodIfaceCacheData) error {
	entry, err := cache.NewPodInterfaceCache(c.cacheCreator, c.launcherPid).IfaceEntry(key)
	if err != nil {
		return err
	}
	return entry.Write(data)
}

// readPreUpgradeState reads from the pre-upgrade virt-handler-local cache path.
// Remove once all virt-launchers have been upgraded and restarted at least once.
func (c *ConfigStateCache) readPreUpgradeState(key string) (*cache.PodIfaceCacheData, error) {
	data, err := cache.ReadLegacyPodInterfaceCache(c.cacheCreator, c.vmiUID, key)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return data, err
}

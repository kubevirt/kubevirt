/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package network

import (
	"errors"
	"fmt"
	"os"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/cache"
)

type ConfigStateCache struct {
	vmiUID                string
	cacheCreator          cacheCreator
	volatilePodIfaceState map[string]cache.PodIfaceState
}

func NewConfigStateCache(vmiUID string, cacheCreator cacheCreator) ConfigStateCache {
	return NewConfigStateCacheWithPodIfaceStateData(vmiUID, cacheCreator, map[string]cache.PodIfaceState{})
}

func NewConfigStateCacheWithPodIfaceStateData(vmiUID string, cacheCreator cacheCreator, volatilePodIfaceState map[string]cache.PodIfaceState) ConfigStateCache {
	return ConfigStateCache{vmiUID, cacheCreator, volatilePodIfaceState}
}

func (c *ConfigStateCache) Read(key string) (cache.PodIfaceState, error) {
	if volatilePodIfaceState, ok := c.volatilePodIfaceState[key]; ok {
		return volatilePodIfaceState, nil
	}
	podIfaceCacheData, err := cache.ReadPodInterfaceCache(c.cacheCreator, c.vmiUID, key)
	var state cache.PodIfaceState
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return defaultState, fmt.Errorf("failed to read pod interface network state from cache: %v", err)
		}
		state = defaultState
	} else {
		state = podIfaceCacheData.State
	}
	c.volatilePodIfaceState[key] = state
	return state, nil
}

func (c *ConfigStateCache) Write(key string, state cache.PodIfaceState) error {
	podIfaceCacheData, err := cache.ReadPodInterfaceCache(c.cacheCreator, c.vmiUID, key)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Log.Reason(err).Errorf("failed to read pod interface network (%s) state from cache", key)
			return err
		}
		podIfaceCacheData = &cache.PodIfaceCacheData{}
	}

	podIfaceCacheData.State = state
	err = cache.WritePodInterfaceCache(c.cacheCreator, c.vmiUID, key, podIfaceCacheData)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to write pod interface network (%s) state to cache", key)
		return err
	}
	c.volatilePodIfaceState[key] = state
	return nil
}

func (c *ConfigStateCache) Delete(key string) error {
	delete(c.volatilePodIfaceState, key)
	podIfaceCacheData, err := cache.ReadPodInterfaceCache(c.cacheCreator, c.vmiUID, key)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	podIfaceCacheData.State = cache.PodIfaceNetworkPreparationPending
	err = cache.WritePodInterfaceCache(c.cacheCreator, c.vmiUID, key, podIfaceCacheData)
	if err != nil {
		return err
	}
	return nil
}

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

package network

import (
	"errors"
	"fmt"
	"os"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/cache"
	neterrors "kubevirt.io/kubevirt/pkg/network/errors"
)

type ConfigState struct {
	cacheCreator cacheCreator
	vmiUID       string
}

func NewConfigState(cc cacheCreator, vmiUID string) ConfigState {
	return ConfigState{cc, vmiUID}
}

// Run passes through the state machine flow, executing the following steps:
// - Discover the current pod network configuration status and persist some of it for future use.
// - Configure the pod network.
//
// The discovery step can be executed repeatedly with no limitation.
// The configuration step is allowed to run only once. Any attempt to run it again will cause a critical error.
func (c ConfigState) Run(podIfaceName string, discoverFunc func() error, configFunc func() error) error {
	state, err := c.read(podIfaceName)
	if err != nil {
		return err
	}

	switch state {
	case cache.PodIfaceNetworkPreparationStarted:
		return neterrors.CreateCriticalNetworkError(
			fmt.Errorf("pod interface %s network preparation cannot be restarted", podIfaceName),
		)
	case cache.PodIfaceNetworkPreparationFinished:
		return nil
	}

	if ferr := discoverFunc(); ferr != nil {
		return ferr
	}

	if werr := c.write(podIfaceName, cache.PodIfaceNetworkPreparationStarted); werr != nil {
		return fmt.Errorf("failed to mark configuration as started for %s: %v", podIfaceName, werr)
	}

	// The discovery step must be called *before* the configuration step, allowing it to persist/cache the
	// original pod network status. The configuration step mutates the pod network.
	if ferr := configFunc(); ferr != nil {
		log.Log.Reason(err).Errorf("failed to configure pod network: %s", podIfaceName)
		return neterrors.CreateCriticalNetworkError(ferr)
	}

	if werr := c.write(podIfaceName, cache.PodIfaceNetworkPreparationFinished); werr != nil {
		return neterrors.CreateCriticalNetworkError(
			fmt.Errorf("failed to mark configuration as finished for %s: %v", podIfaceName, werr),
		)
	}
	return nil
}

func (c ConfigState) read(networkName string) (cache.PodIfaceState, error) {
	var podIfaceCacheData *cache.PodIfaceCacheData
	podIfaceCacheData, err := cache.ReadPodInterfaceCache(c.cacheCreator, c.vmiUID, networkName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaultState, nil
		}
		return defaultState, fmt.Errorf("failed to read pod interface network state from cache: %v", err)
	}
	return podIfaceCacheData.State, nil
}

func (c ConfigState) write(networkName string, state cache.PodIfaceState) error {
	var podIfaceCacheData *cache.PodIfaceCacheData
	podIfaceCacheData, err := cache.ReadPodInterfaceCache(c.cacheCreator, c.vmiUID, networkName)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Log.Reason(err).Errorf("failed to read pod interface network (%s) state from cache", networkName)
			return err
		}
		podIfaceCacheData = &cache.PodIfaceCacheData{}
	}

	podIfaceCacheData.State = state
	err = cache.WritePodInterfaceCache(c.cacheCreator, c.vmiUID, networkName, podIfaceCacheData)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to write pod interface network (%s) state to cache", networkName)
		return err
	}
	return nil
}

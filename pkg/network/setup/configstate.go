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
	v1 "kubevirt.io/api/core/v1"

	"k8s.io/apimachinery/pkg/util/errors"

	"kubevirt.io/kubevirt/pkg/network/cache"
)

type configStateCacheRUD interface {
	Read(networkName string) (cache.PodIfaceState, error)
	Write(networkName string, state cache.PodIfaceState) error
	Delete(networkName string) error
}

type ConfigState struct {
	cache configStateCacheRUD
	ns    NSExecutor
}

func NewConfigState(configStateCache configStateCacheRUD, ns NSExecutor) ConfigState {
	return ConfigState{cache: configStateCache, ns: ns}
}

func (c *ConfigState) Unplug(networks []v1.Network, filterFunc func([]v1.Network) ([]string, error), cleanupFunc func(string) error) error {
	var nonPendingNetworks []v1.Network
	var err error
	if nonPendingNetworks, err = c.nonPendingNetworks(networks); err != nil {
		return err
	}

	if len(nonPendingNetworks) == 0 {
		return nil
	}
	err = c.ns.Do(func() error {
		networksToUnplug, doErr := filterFunc(nonPendingNetworks)
		if doErr != nil {
			return doErr
		}

		var cleanupErrors []error
		for _, net := range networksToUnplug {
			if cleanupErr := cleanupFunc(net); cleanupErr != nil {
				cleanupErrors = append(cleanupErrors, cleanupErr)
			} else if cleanupErr := c.cache.Delete(net); cleanupErr != nil {
				cleanupErrors = append(cleanupErrors, cleanupErr)
			}
		}
		return errors.NewAggregate(cleanupErrors)
	})
	return err
}

func (c *ConfigState) nonPendingNetworks(networks []v1.Network) ([]v1.Network, error) {
	var nonPendingNetworks []v1.Network

	for _, net := range networks {

		state, err := c.cache.Read(net.Name)
		if err != nil {
			return nil, err
		}
		if state != cache.PodIfaceNetworkPreparationPending {
			nonPendingNetworks = append(nonPendingNetworks, net)
		}
	}
	return nonPendingNetworks, nil
}

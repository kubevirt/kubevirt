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

package cache

import (
	v1 "kubevirt.io/api/core/v1"
)

type PodIfaceState int

const (
	PodIfaceNetworkPreparationPending PodIfaceState = iota
	PodIfaceNetworkPreparationStarted
	PodIfaceNetworkPreparationFinished
)

type PodCacheInterface struct {
	Iface  *v1.Interface `json:"iface,omitempty"`
	PodIP  string        `json:"podIP,omitempty"`
	PodIPs []string      `json:"podIPs,omitempty"`
	State  PodIfaceState `json:"networkState,omitempty"`
}

type PodInterfaceCache struct {
	cache *Cache
}

func PodInterfaceCachePath(uid string) string {
	return getInterfaceCacheFile(virtHandlerCachePattern, uid, "")
}

func NewPodInterfaceCache(cache *Cache) PodInterfaceCache {
	return PodInterfaceCache{cache: cache}
}

func (p PodInterfaceCache) IfaceEntry(ifaceName string) (PodInterfaceCache, error) {
	cache, err := p.cache.Entry(ifaceName)
	if err != nil {
		return PodInterfaceCache{}, err
	}

	return NewPodInterfaceCache(&cache), nil
}

func (p PodInterfaceCache) Read() (*PodCacheInterface, error) {
	iface := &PodCacheInterface{}
	_, err := p.cache.Read(iface)
	return iface, err
}

func (p PodInterfaceCache) Write(cacheInterface *PodCacheInterface) error {
	return p.cache.Write(cacheInterface)
}

func (p PodInterfaceCache) Remove() error {
	return p.cache.Delete()
}

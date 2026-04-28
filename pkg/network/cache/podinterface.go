/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package cache

import (
	"path/filepath"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/util"
)

type PodIfaceState int

const (
	PodIfaceNetworkPreparationPending PodIfaceState = iota
	PodIfaceNetworkPreparationStarted
	PodIfaceNetworkPreparationFinished
)

type PodIfaceCacheData struct {
	Iface  *v1.Interface `json:"iface,omitempty"`
	PodIP  string        `json:"podIP,omitempty"`
	PodIPs []string      `json:"podIPs,omitempty"`
	State  PodIfaceState `json:"networkState,omitempty"`
}

type PodInterfaceCache struct {
	cache *Cache
}

func ReadPodInterfaceCache(c cacheCreator, uid, ifaceName string) (*PodIfaceCacheData, error) {
	podCache, err := NewPodInterfaceCache(c, uid).IfaceEntry(ifaceName)
	if err != nil {
		return nil, err
	}
	return podCache.Read()
}

func WritePodInterfaceCache(c cacheCreator, uid, ifaceName string, cacheInterface *PodIfaceCacheData) error {
	podCache, err := NewPodInterfaceCache(c, uid).IfaceEntry(ifaceName)
	if err != nil {
		return err
	}
	return podCache.Write(cacheInterface)
}

func DeletePodInterfaceCache(c cacheCreator, uid, ifaceName string) error {
	podCache, err := NewPodInterfaceCache(c, uid).IfaceEntry(ifaceName)
	if err != nil {
		return err
	}
	return podCache.Remove()
}

func NewPodInterfaceCache(creator cacheCreator, uid string) PodInterfaceCache {
	const podIfaceCacheDirName = "network-info-cache"
	return PodInterfaceCache{creator.New(filepath.Join(util.VirtPrivateDir, podIfaceCacheDirName, uid))}
}

func (p PodInterfaceCache) IfaceEntry(ifaceName string) (PodInterfaceCache, error) {
	cache, err := p.cache.Entry(ifaceName)
	if err != nil {
		return PodInterfaceCache{}, err
	}

	return PodInterfaceCache{&cache}, nil
}

func (p PodInterfaceCache) Read() (*PodIfaceCacheData, error) {
	iface := &PodIfaceCacheData{}
	_, err := p.cache.Read(iface)
	return iface, err
}

func (p PodInterfaceCache) Write(cacheInterface *PodIfaceCacheData) error {
	return p.cache.Write(cacheInterface)
}

func (p PodInterfaceCache) Remove() error {
	return p.cache.Delete()
}

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
	"fmt"
	"path/filepath"

	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type DomainInterfaceCache struct {
	cache *Cache
}

func ReadDomainInterfaceCache(c cacheCreator, pid, ifaceName string) (*api.Interface, error) {
	domainCache, err := NewDomainInterfaceCache(c, pid).IfaceEntry(ifaceName)
	if err != nil {
		return nil, err
	}
	return domainCache.Read()
}

func WriteDomainInterfaceCache(c cacheCreator, pid, ifaceName string, domainInterface *api.Interface) error {
	domainCache, err := NewDomainInterfaceCache(c, pid).IfaceEntry(ifaceName)
	if err != nil {
		return err
	}
	return domainCache.Write(domainInterface)
}

func NewDomainInterfaceCache(creator cacheCreator, pid string) DomainInterfaceCache {
	podRootFilesystemPath := fmt.Sprintf("/proc/%s/root", pid)
	return DomainInterfaceCache{creator.New(filepath.Join(podRootFilesystemPath, util.VirtPrivateDir))}
}

func DeleteDomainInterfaceCache(c cacheCreator, pid, ifaceName string) error {
	domainCache, err := NewDomainInterfaceCache(c, pid).IfaceEntry(ifaceName)
	if err != nil {
		return err
	}
	return domainCache.Delete()
}

func (d DomainInterfaceCache) IfaceEntry(ifaceName string) (DomainInterfaceCache, error) {
	const domainIfaceCacheFileFormat = "interface-cache-%s.json"
	cacheFileName := fmt.Sprintf(domainIfaceCacheFileFormat, ifaceName)
	cache, err := d.cache.Entry(cacheFileName)
	if err != nil {
		return DomainInterfaceCache{}, err
	}

	return DomainInterfaceCache{&cache}, nil
}

func (d DomainInterfaceCache) Read() (*api.Interface, error) {
	iface := &api.Interface{}
	_, err := d.cache.Read(iface)
	return iface, err
}

func (d DomainInterfaceCache) Write(domainInterface *api.Interface) error {
	return d.cache.Write(domainInterface)
}

func (d DomainInterfaceCache) Delete() error {
	return d.cache.Delete()
}

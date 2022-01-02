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
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const virtLauncherCachedPattern = "/proc/%s/root/var/run/kubevirt-private/interface-cache-%s.json"

type DomainInterfaceCache struct {
	cache *Cache
}

func NewDomainInterfaceCache(creator cacheCreator, pid string) DomainInterfaceCache {
	return DomainInterfaceCache{creator.New(DomainInterfaceCachePath(pid))}
}

func DomainInterfaceCachePath(pid string) string {
	return getInterfaceCacheFile(virtLauncherCachedPattern, pid, "")
}

func (d DomainInterfaceCache) IfaceEntry(ifaceName string) (DomainInterfaceCache, error) {
	cache, err := d.cache.Entry(ifaceName)
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

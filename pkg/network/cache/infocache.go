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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package cache

import (
	"kubevirt.io/kubevirt/pkg/os/fs"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type InterfaceCacheFactory interface {
	CacheDomainInterfaceForPID(pid string) DomainInterfaceStore
	CacheDHCPConfigForPid(pid string) DHCPConfigStore
}

func NewInterfaceCacheFactory() *interfaceCacheFactory {
	return &interfaceCacheFactory{
		fs: fs.New(),
	}
}

func NewInterfaceCacheFactoryWithBasePath(rootPath string) *interfaceCacheFactory {
	return &interfaceCacheFactory{
		fs: fs.NewWithRootPath(rootPath),
	}
}

type interfaceCacheFactory struct {
	fs fs.Fs
}

func (i *interfaceCacheFactory) CacheDomainInterfaceForPID(pid string) DomainInterfaceStore {
	return domainInterfaceStore{pid: pid, fs: i.fs, pattern: virtLauncherCachedPattern}
}

func (i *interfaceCacheFactory) CacheDHCPConfigForPid(pid string) DHCPConfigStore {
	return dhcpConfigCacheStore{pid: pid, fs: i.fs, pattern: dhcpConfigCachedPattern}
}

type DomainInterfaceStore interface {
	Read(ifaceName string) (*api.Interface, error)
	Write(ifaceName string, cacheInterface *api.Interface) error
}

type DHCPConfigStore interface {
	Read(ifaceName string) (*DHCPConfig, error)
	Write(ifaceName string, cacheInterface *DHCPConfig) error
}

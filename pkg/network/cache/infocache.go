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
	"encoding/json"
	"fmt"
	"path/filepath"

	dutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"

	"kubevirt.io/kubevirt/pkg/os/fs"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const virtHandlerCachePattern = util.VirtPrivateDir + "/network-info-cache/%s/%s"

var virtLauncherCachedPattern = "/proc/%s/root/var/run/kubevirt-private/interface-cache-%s.json"
var dhcpConfigCachedPattern = "/proc/%s/root/var/run/kubevirt-private/vif-cache-%s.json"

type InterfaceCacheFactory interface {
	CacheForVMI(uid string) PodInterfaceCacheStore
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

func (i *interfaceCacheFactory) CacheForVMI(uid string) PodInterfaceCacheStore {
	var cache CacheCreator
	baseCache := cache.New(PodInterfaceCachePath(uid))
	// TODO: Remove this hack when the usage of the all-in-one `interfaceCacheFactory` is removed.
	baseCache.fs = i.fs
	return NewPodInterfaceCache(baseCache)
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

type domainInterfaceStore struct {
	pid     string
	pattern string
	fs      fs.Fs
}

func (d domainInterfaceStore) Read(ifaceName string) (*api.Interface, error) {
	iface := &api.Interface{}
	err := readFromCachedFile(d.fs, iface, getInterfaceCacheFile(d.pattern, d.pid, ifaceName))
	return iface, err
}

func (d domainInterfaceStore) Write(ifaceName string, cacheInterface *api.Interface) (err error) {
	err = writeToCachedFile(d.fs, cacheInterface, getInterfaceCacheFile(d.pattern, d.pid, ifaceName))
	return
}

type dhcpConfigCacheStore struct {
	pid     string
	pattern string
	fs      fs.Fs
}

func (d dhcpConfigCacheStore) Read(ifaceName string) (*DHCPConfig, error) {
	cachedIface := &DHCPConfig{}
	err := readFromCachedFile(d.fs, cachedIface, d.getInterfaceCacheFile(ifaceName))
	return cachedIface, err
}

func (d dhcpConfigCacheStore) Write(ifaceName string, ifaceToCache *DHCPConfig) error {
	return writeToCachedFile(d.fs, ifaceToCache, d.getInterfaceCacheFile(ifaceName))
}

func (d dhcpConfigCacheStore) getInterfaceCacheFile(ifaceName string) string {
	return getInterfaceCacheFile(d.pattern, d.pid, ifaceName)
}

func writeToCachedFile(fs cacheFS, obj interface{}, fileName string) error {
	if err := fs.MkdirAll(filepath.Dir(fileName), 0750); err != nil {
		return err
	}
	buf, err := json.MarshalIndent(&obj, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling cached object: %v", err)
	}

	err = fs.WriteFile(fileName, buf, 0604)
	if err != nil {
		return fmt.Errorf("error writing cached object: %v", err)
	}
	return dutils.DefaultOwnershipManager.SetFileOwnership(fileName)
}

func readFromCachedFile(fs cacheFS, obj interface{}, fileName string) error {
	buf, err := fs.ReadFile(fileName)
	if err != nil {
		return err
	}

	err = json.Unmarshal(buf, &obj)
	if err != nil {
		return fmt.Errorf("error unmarshaling cached object: %v", err)
	}
	return nil
}

func getInterfaceCacheFile(pattern, id, name string) string {
	return filepath.Join(fmt.Sprintf(pattern, id, name))
}

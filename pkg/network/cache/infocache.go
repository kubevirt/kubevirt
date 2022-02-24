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

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/os/fs"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const networkInfoDir = util.VirtPrivateDir + "/network-info-cache"
const virtHandlerCachePattern = networkInfoDir + "/%s/%s"

var virtLauncherCachedPattern = "/proc/%s/root/var/run/kubevirt-private/interface-cache-%s.json"
var dhcpConfigCachedPattern = "/proc/%s/root/var/run/kubevirt-private/vif-cache-%s.json"

type InterfaceCacheFactory interface {
	CacheForVMI(vmi *v1.VirtualMachineInstance) PodInterfaceCacheStore
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

func (i *interfaceCacheFactory) CacheForVMI(vmi *v1.VirtualMachineInstance) PodInterfaceCacheStore {
	return newPodInterfaceCacheStore(vmi, i.fs, virtHandlerCachePattern)
}

func (i *interfaceCacheFactory) CacheDomainInterfaceForPID(pid string) DomainInterfaceStore {
	return newDomainInterfaceStore(pid, i.fs, virtLauncherCachedPattern)
}

func (i *interfaceCacheFactory) CacheDHCPConfigForPid(pid string) DHCPConfigStore {
	return newDHCPConfigCacheStore(pid, i.fs, dhcpConfigCachedPattern)
}

type DomainInterfaceStore interface {
	Read(ifaceName string) (*api.Interface, error)
	Write(ifaceName string, cacheInterface *api.Interface) error
}

type PodInterfaceCacheStore interface {
	Read(ifaceName string) (*PodCacheInterface, error)
	Write(ifaceName string, cacheInterface *PodCacheInterface) error
	Remove() error
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

func newDomainInterfaceStore(pid string, fs fs.Fs, pattern string) DomainInterfaceStore {
	return domainInterfaceStore{pid: pid, fs: fs, pattern: pattern}
}

type podInterfaceCacheStore struct {
	vmi     *v1.VirtualMachineInstance
	pattern string
	fs      fs.Fs
}

func (p podInterfaceCacheStore) Read(ifaceName string) (*PodCacheInterface, error) {
	iface := &PodCacheInterface{}
	err := readFromCachedFile(p.fs, iface, getInterfaceCacheFile(p.pattern, string(p.vmi.UID), ifaceName))
	return iface, err
}

func (p podInterfaceCacheStore) Write(iface string, cacheInterface *PodCacheInterface) (err error) {
	err = writeToCachedFile(p.fs, cacheInterface, getInterfaceCacheFile(p.pattern, string(p.vmi.UID), iface))
	return
}

func (p podInterfaceCacheStore) Remove() error {
	return p.fs.RemoveAll(filepath.Join(networkInfoDir, string(p.vmi.UID)))
}

func newPodInterfaceCacheStore(vmi *v1.VirtualMachineInstance, fs fs.Fs, pattern string) PodInterfaceCacheStore {
	return podInterfaceCacheStore{vmi: vmi, fs: fs, pattern: pattern}
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

func newDHCPConfigCacheStore(pid string, fs fs.Fs, pattern string) dhcpConfigCacheStore {
	return dhcpConfigCacheStore{pid: pid, fs: fs, pattern: pattern}
}

func writeToCachedFile(fs fs.Fs, obj interface{}, fileName string) error {
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

func readFromCachedFile(fs fs.Fs, obj interface{}, fileName string) error {
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

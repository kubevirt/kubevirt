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
	"io/ioutil"
	"os"
	"path/filepath"

	dutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"

	v1 "kubevirt.io/client-go/api/v1"
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
	return &interfaceCacheFactory{}
}

type interfaceCacheFactory struct {
	baseDir string
}

func (i *interfaceCacheFactory) CacheForVMI(vmi *v1.VirtualMachineInstance) PodInterfaceCacheStore {
	return newPodInterfaceCacheStore(vmi, i.baseDir, virtHandlerCachePattern)
}

func (i *interfaceCacheFactory) CacheDomainInterfaceForPID(pid string) DomainInterfaceStore {
	return newDomainInterfaceStore(pid, i.baseDir, virtLauncherCachedPattern)
}

func (i *interfaceCacheFactory) CacheDHCPConfigForPid(pid string) DHCPConfigStore {
	return newDHCPConfigCacheStore(pid, i.baseDir, dhcpConfigCachedPattern)
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
	baseDir string
}

func (d domainInterfaceStore) Read(ifaceName string) (*api.Interface, error) {
	iface := &api.Interface{}
	err := readFromCachedFile(iface, getInterfaceCacheFile(d.baseDir, d.pattern, d.pid, ifaceName))
	return iface, err
}

func (d domainInterfaceStore) Write(ifaceName string, cacheInterface *api.Interface) (err error) {
	err = writeToCachedFile(cacheInterface, getInterfaceCacheFile(d.baseDir, d.pattern, d.pid, ifaceName))
	return
}

func newDomainInterfaceStore(pid string, baseDir, pattern string) DomainInterfaceStore {
	return domainInterfaceStore{pid: pid, baseDir: baseDir, pattern: pattern}
}

type podInterfaceCacheStore struct {
	vmi     *v1.VirtualMachineInstance
	pattern string
	baseDir string
}

func (p podInterfaceCacheStore) Read(ifaceName string) (*PodCacheInterface, error) {
	iface := &PodCacheInterface{}
	err := readFromCachedFile(iface, getInterfaceCacheFile(p.baseDir, p.pattern, string(p.vmi.UID), ifaceName))
	return iface, err
}

func (p podInterfaceCacheStore) Write(iface string, cacheInterface *PodCacheInterface) (err error) {
	err = writeToCachedFile(cacheInterface, getInterfaceCacheFile(p.baseDir, p.pattern, string(p.vmi.UID), iface))
	return
}

func (p podInterfaceCacheStore) Remove() error {
	return os.RemoveAll(filepath.Join(p.baseDir, networkInfoDir, string(p.vmi.UID)))
}

func newPodInterfaceCacheStore(vmi *v1.VirtualMachineInstance, baseDir, pattern string) PodInterfaceCacheStore {
	return podInterfaceCacheStore{vmi: vmi, baseDir: baseDir, pattern: pattern}
}

type dhcpConfigCacheStore struct {
	pid     string
	pattern string
	baseDir string
}

func (d dhcpConfigCacheStore) Read(ifaceName string) (*DHCPConfig, error) {
	cachedIface := &DHCPConfig{}
	err := readFromCachedFile(cachedIface, d.getInterfaceCacheFile(ifaceName))
	return cachedIface, err
}

func (d dhcpConfigCacheStore) Write(ifaceName string, ifaceToCache *DHCPConfig) error {
	return writeToCachedFile(ifaceToCache, d.getInterfaceCacheFile(ifaceName))
}

func (d dhcpConfigCacheStore) getInterfaceCacheFile(ifaceName string) string {
	return getInterfaceCacheFile(d.baseDir, d.pattern, d.pid, ifaceName)
}

func newDHCPConfigCacheStore(pid string, baseDir, pattern string) dhcpConfigCacheStore {
	return dhcpConfigCacheStore{pid: pid, baseDir: baseDir, pattern: pattern}
}

func writeToCachedFile(obj interface{}, fileName string) error {
	if err := os.MkdirAll(filepath.Dir(fileName), 0750); err != nil {
		return err
	}
	buf, err := json.MarshalIndent(&obj, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling cached object: %v", err)
	}

	err = ioutil.WriteFile(fileName, buf, 0604)
	if err != nil {
		return fmt.Errorf("error writing cached object: %v", err)
	}
	return dutils.DefaultOwnershipManager.SetFileOwnership(fileName)
}

func readFromCachedFile(obj interface{}, fileName string) error {
	buf, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}

	err = json.Unmarshal(buf, &obj)
	if err != nil {
		return fmt.Errorf("error unmarshaling cached object: %v", err)
	}
	return nil
}

func getInterfaceCacheFile(baseDir, pattern, id, name string) string {
	return filepath.Join(baseDir, fmt.Sprintf(pattern, id, name))
}

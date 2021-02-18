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

package network

/*
 ATTENTION: Rerun code generators when interface signatures are modified.
*/

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const networkInfoDir = util.VirtPrivateDir + "/network-info-cache"
const virtHandlerCachePattern = networkInfoDir + "/%s/%s"

var virtLauncherCachedPattern = "/proc/%s/root/var/run/kubevirt-private/interface-cache-%s.json"

type InterfaceCacheFactory interface {
	CacheForVMI(vmi *v1.VirtualMachineInstance) PodInterfaceCacheStore
	CacheForPID(pid string) DomainInterfaceStore
}

func NewInterfaceCacheFactory() InterfaceCacheFactory {
	return &interfaceCacheFactory{}
}

type interfaceCacheFactory struct {
}

func (i *interfaceCacheFactory) CacheForVMI(vmi *v1.VirtualMachineInstance) PodInterfaceCacheStore {
	return NewPodInterfaceCacheStore(vmi)
}

func (i *interfaceCacheFactory) CacheForPID(pid string) DomainInterfaceStore {
	return NewDomainInterfaceStore(pid)
}

type DomainInterfaceStore interface {
	Read(iface string) (*api.Interface, error)
	Write(iface string, cacheInterface *api.Interface) error
}

type PodInterfaceCacheStore interface {
	Read(iface string) (*PodCacheInterface, error)
	Write(iface string, cacheInterface *PodCacheInterface) error
	Remove() error
}

type domainInterfaceStore struct {
	pid string
}

func (d domainInterfaceStore) Read(iface string) (file *api.Interface, err error) {
	file = &api.Interface{}
	err = readFromVirtLauncherCachedFile(file, d.pid, iface)
	return
}

func (d domainInterfaceStore) Write(iface string, cacheInterface *api.Interface) (err error) {
	err = writeToVirtLauncherCachedFile(cacheInterface, d.pid, iface)
	return
}

func NewDomainInterfaceStore(pid string) DomainInterfaceStore {
	return domainInterfaceStore{pid: pid}
}

type podInterfaceCacheStore struct {
	vmi *v1.VirtualMachineInstance
}

func (p podInterfaceCacheStore) Read(iface string) (file *PodCacheInterface, err error) {
	file = &PodCacheInterface{}
	err = readFromVirtHandlerCachedFil(file, p.vmi.UID, iface)
	return
}

func (p podInterfaceCacheStore) Write(iface string, cacheInterface *PodCacheInterface) (err error) {
	err = writeToVirtHandlerCachedFil(cacheInterface, p.vmi.UID, iface)
	return
}

func (p podInterfaceCacheStore) Remove() error {
	return os.RemoveAll(filepath.Join(networkInfoDir, string(p.vmi.UID)))
}

func NewPodInterfaceCacheStore(vmi *v1.VirtualMachineInstance) PodInterfaceCacheStore {
	return podInterfaceCacheStore{vmi: vmi}
}

func writeToCachedFile(obj interface{}, fileName string) error {
	buf, err := json.MarshalIndent(&obj, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling cached object: %v", err)
	}

	err = ioutil.WriteFile(fileName, buf, 0644)
	if err != nil {
		return fmt.Errorf("error writing cached object: %v", err)
	}
	return nil
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

func readFromVirtLauncherCachedFile(obj interface{}, pid, ifaceName string) error {
	fileName := getInterfaceCacheFile(virtLauncherCachedPattern, pid, ifaceName)
	return readFromCachedFile(obj, fileName)
}

func writeToVirtLauncherCachedFile(obj interface{}, pid, ifaceName string) error {
	fileName := getInterfaceCacheFile(virtLauncherCachedPattern, pid, ifaceName)
	if err := os.MkdirAll(filepath.Dir(fileName), 0755); err != nil {
		return err
	}
	return writeToCachedFile(obj, fileName)
}

func readFromVirtHandlerCachedFil(obj interface{}, vmiuid types.UID, ifaceName string) error {
	fileName := getInterfaceCacheFile(virtHandlerCachePattern, string(vmiuid), ifaceName)
	return readFromCachedFile(obj, fileName)
}

func writeToVirtHandlerCachedFil(obj interface{}, vmiuid types.UID, ifaceName string) error {
	fileName := getInterfaceCacheFile(virtHandlerCachePattern, string(vmiuid), ifaceName)
	if err := os.MkdirAll(filepath.Dir(fileName), 0755); err != nil {
		return err
	}
	return writeToCachedFile(obj, fileName)
}

func getInterfaceCacheFile(pattern, id, name string) string {
	return fmt.Sprintf(pattern, id, name)
}

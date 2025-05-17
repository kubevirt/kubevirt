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
 * Copyright The KubeVirt Authors.
 */

package network

import (
	"os"
	"sync"
	"testing"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/network/cache"
	kfs "kubevirt.io/kubevirt/pkg/os/fs"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/testutils"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func TestNetwork(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}

func newVMIBridgeInterface(namespace string, name string) *v1.VirtualMachineInstance {
	vmi := libvmi.New(
		libvmi.WithNamespace(namespace),
		libvmi.WithName(name),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
	)
	v1.SetObjectDefaults_VirtualMachineInstance(vmi)
	return vmi
}

func NewDomainWithBridgeInterface() *api.Domain {
	domain := &api.Domain{}
	domain.Spec.Devices.Interfaces = []api.Interface{{
		Model: &api.Model{
			Type: v1.VirtIO,
		},
		Type: "bridge",
		Source: api.InterfaceSource{
			Bridge: api.DefaultBridgeName,
		},
		Alias: api.NewUserDefinedAlias("default"),
	},
	}
	return domain
}

type tempCacheCreator struct {
	once   sync.Once
	tmpDir string
}

func (c *tempCacheCreator) New(filePath string) *cache.Cache {
	c.once.Do(func() {
		tmpDir, err := os.MkdirTemp("", "temp-cache")
		if err != nil {
			panic("Unable to create temp cache directory")
		}
		c.tmpDir = tmpDir
	})
	return cache.NewCustomCache(filePath, kfs.NewWithRootPath(c.tmpDir))
}

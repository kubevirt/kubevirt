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

package domainspec

import (
	"testing"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/testutils"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func TestDomainSpec(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}

func newVMIMasqueradeInterface(namespace, vmiName string) *v1.VirtualMachineInstance {
	return libvmi.New(
		libvmi.WithNamespace(namespace),
		libvmi.WithName(vmiName),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithInterface(*v1.DefaultMasqueradeNetworkInterface()),
	)
}

func NewDomainInterface(name string) *api.Domain {
	domain := &api.Domain{}
	domain.Spec.Devices.Interfaces = []api.Interface{{
		Alias: api.NewUserDefinedAlias(name),
		Model: &api.Model{
			Type: v1.VirtIO,
		},
		Type: "ethernet",
	}}
	return domain
}

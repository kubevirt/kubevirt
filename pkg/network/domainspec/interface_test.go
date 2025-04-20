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
 *
 */

package domainspec_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/network/domainspec"
)

var _ = Describe("VMI interfaces", func() {
	const (
		binding1              = "binding1"
		binding2              = "binding2"
		binding3              = "binding3"
		iface1                = "iface1"
		iface2                = "iface2"
		iface3                = "iface3"
		iface4                = "iface4"
		iface5                = "iface5"
		iface6                = "iface6"
		iface7                = "iface7"
		otherDoaminAttachemnt = "otherAttachment"
	)

	var networkBindings map[string]v1.InterfaceBindingPlugin
	var vmiSpecIfaces []v1.Interface

	BeforeEach(func() {
		networkBindings = map[string]v1.InterfaceBindingPlugin{
			binding1: {
				DomainAttachmentType: v1.Tap,
				Migration:            &v1.InterfaceBindingMigration{},
			},
			binding2: {
				DomainAttachmentType: otherDoaminAttachemnt,
			},
			binding3: {
				Migration: &v1.InterfaceBindingMigration{
					Method: v1.LinkRefresh,
				},
			},
		}
	})

	vmiSpecIfaces = []v1.Interface{
		{
			Name:                   iface1,
			InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
		},
		{
			Name:                   iface2,
			InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
		},
		{
			Name:                   iface3,
			InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}},
		},
		{
			// Macvtap is removed in v1.3. This scenario is tracking old VMIs that are still processed in the reconcile loop.
			Name:                   iface4,
			InterfaceBindingMethod: v1.InterfaceBindingMethod{DeprecatedMacvtap: &v1.DeprecatedInterfaceMacvtap{}},
		},
		{
			Name:    iface5,
			Binding: &v1.PluginBinding{Name: binding1},
		},
		{
			Name:    iface6,
			Binding: &v1.PluginBinding{Name: binding2},
		},
		{
			Name:    iface7,
			Binding: &v1.PluginBinding{Name: binding3},
		},
	}

	Context("DomainAttachmentByInterfaceName", func() {
		It("should return the correct mapping", func() {
			expectedMap := map[string]string{
				iface1: string(v1.Tap),
				iface2: string(v1.Tap),
				iface4: string(v1.Tap),
				iface5: string(v1.Tap),
				iface6: otherDoaminAttachemnt,
			}
			Expect(domainspec.DomainAttachmentByInterfaceName(vmiSpecIfaces, networkBindings)).To(Equal(expectedMap))
		})

		It("should consider a managedTap type as a tap type", func() {
			vmiIfaces := []v1.Interface{{Name: iface1, Binding: &v1.PluginBinding{Name: binding1}}}
			netBindings := map[string]v1.InterfaceBindingPlugin{binding1: {DomainAttachmentType: v1.ManagedTap}}
			Expect(domainspec.DomainAttachmentByInterfaceName(vmiIfaces, netBindings)).To(Equal(
				map[string]string{iface1: string(v1.Tap)},
			))
		})
	})

	Context("BindingMigrationByInterfaceName", func() {
		It("should return the correct mapping", func() {
			expectedMap := map[string]*cmdv1.InterfaceBindingMigration{
				iface5: {},
				iface7: {
					Method: string(v1.LinkRefresh),
				},
			}
			Expect(domainspec.BindingMigrationByInterfaceName(vmiSpecIfaces, networkBindings)).To(Equal(expectedMap))
		})
	})
})

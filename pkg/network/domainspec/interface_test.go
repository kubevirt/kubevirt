/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
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
			Name:    iface4,
			Binding: &v1.PluginBinding{Name: binding1},
		},
		{
			Name:    iface5,
			Binding: &v1.PluginBinding{Name: binding2},
		},
		{
			Name:    iface6,
			Binding: &v1.PluginBinding{Name: binding3},
		},
	}

	Context("DomainAttachmentByInterfaceName", func() {
		It("should return the correct mapping", func() {
			expectedMap := map[string]string{
				iface1: string(v1.Tap),
				iface2: string(v1.Tap),
				iface4: string(v1.Tap),
				iface5: otherDoaminAttachemnt,
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
				iface4: {},
				iface6: {
					Method: string(v1.LinkRefresh),
				},
			}
			Expect(domainspec.BindingMigrationByInterfaceName(vmiSpecIfaces, networkBindings)).To(Equal(expectedMap))
		})
	})
})

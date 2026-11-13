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

package compute_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute"
)

var _ = Describe("IOThreads Configurator", func() {
	DescribeTable("should configure domain IOThreads from the VMI", func(
		vmi *v1.VirtualMachineInstance,
		scsiMultiIOEnabled bool,
		multiIOAutoEnabled bool,
		expectedThreads *api.IOThreads,
	) {
		var domain api.Domain
		Expect(compute.NewIOThreadsDomainConfigurator(
			compute.IOThreadsWithSCSIMultiIOThreadEnabled(scsiMultiIOEnabled),
			compute.IOThreadsWithMultiIOThreadAutoPolicyEnabled(multiIOAutoEnabled),
		).Configure(vmi, &domain)).To(Succeed())
		Expect(domain.Spec.IOThreads).To(Equal(expectedThreads))
	},
		Entry("when the VMI has no IOThreads",
			libvmi.New(),
			false, false,
			nil,
		),
		Entry("when auto policy is set",
			libvmi.New(
				libvmi.WithIOThreadsPolicy(v1.IOThreadsPolicyAuto),
				libvmi.WithCPURequest("2"),
				libvmi.WithDisk("disk0", v1.DiskBusVirtio),
				libvmi.WithDisk("disk1", v1.DiskBusVirtio),
			),
			false, false,
			&api.IOThreads{IOThreads: 2},
		),
		Entry("when supplemental pool policy is set",
			libvmi.New(
				libvmi.WithIOThreadsPolicy(v1.IOThreadsPolicySupplementalPool),
				libvmi.WithIOThreads(v1.DiskIOThreads{SupplementalPoolThreadCount: new(uint32(5))}),
			),
			false, false,
			&api.IOThreads{IOThreads: 5},
		),
		Entry("when multi-IO auto policy caps the auto pool",
			libvmi.New(
				libvmi.WithIOThreadsPolicy(v1.IOThreadsPolicyAuto),
				libvmi.WithCPURequest("2"),
				libvmi.WithDisk("disk0", v1.DiskBusVirtio),
				libvmi.WithDisk("disk1", v1.DiskBusVirtio),
				libvmi.WithDisk("disk2", v1.DiskBusVirtio),
				libvmi.WithDisk("dedicated", v1.DiskBusVirtio, libvmi.WithDedicatedIOThreads(true)),
			),
			true, true,
			// auto pool will be capped to 2 but then 1 additional thread will be added for the dedicated disk
			&api.IOThreads{IOThreads: 3},
		),
	)
})

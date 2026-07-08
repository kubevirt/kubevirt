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

var _ = Describe("MemoryBackingConfigurator", func() {
	const memfdSupported = true

	DescribeTable("should configure memory backing",
		func(vmi *v1.VirtualMachineInstance, isMemfdSupported bool, expectedMemoryBacking *api.MemoryBacking) {
			var domain api.Domain

			configurator := compute.NewMemoryBackingConfigurator(isMemfdSupported)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{Spec: api.DomainSpec{
				MemoryBacking: expectedMemoryBacking,
			}}
			Expect(domain).To(Equal(expectedDomain))
		},
		Entry("no hugepages, virtiofs, or passt",
			libvmi.New(),
			memfdSupported,
			nil,
		),
		Entry("hugepages",
			libvmi.New(libvmi.WithHugepages("2Mi")),
			memfdSupported,
			&api.MemoryBacking{
				HugePages: &api.HugePages{},
				Source:    &api.MemoryBackingSource{Type: "memfd"},
			},
		),
		Entry("hugepages with memfd annotation false",
			libvmi.New(
				libvmi.WithHugepages("2Mi"),
				libvmi.WithAnnotation(v1.MemfdMemoryBackend, "false"),
			),
			memfdSupported,
			&api.MemoryBacking{
				HugePages: &api.HugePages{},
			},
		),
		Entry("virtiofs",
			libvmi.New(libvmi.WithFilesystemPVC("test-pvc")),
			memfdSupported,
			&api.MemoryBacking{
				Access: &api.MemoryBackingAccess{Mode: "shared"},
				Source: &api.MemoryBackingSource{Type: "memfd"},
			},
		),
		Entry("passt",
			libvmi.New(libvmi.WithInterface(libvmi.InterfaceDeviceWithPasstBinding("default"))),
			memfdSupported,
			&api.MemoryBacking{
				Access: &api.MemoryBackingAccess{Mode: "shared"},
				Source: &api.MemoryBackingSource{Type: "memfd"},
			},
		),
		Entry("hugepages with virtiofs",
			libvmi.New(
				libvmi.WithHugepages("2Mi"),
				libvmi.WithFilesystemPVC("test-pvc"),
			),
			memfdSupported,
			&api.MemoryBacking{
				HugePages: &api.HugePages{},
				Access:    &api.MemoryBackingAccess{Mode: "shared"},
				Source:    &api.MemoryBackingSource{Type: "memfd"},
			},
		),
		Entry("hugepages without memfd support",
			libvmi.New(libvmi.WithHugepages("2Mi")),
			!memfdSupported,
			&api.MemoryBacking{
				HugePages: &api.HugePages{},
			},
		),
		Entry("virtiofs without memfd support",
			libvmi.New(libvmi.WithFilesystemPVC("test-pvc")),
			!memfdSupported,
			&api.MemoryBacking{
				Access: &api.MemoryBackingAccess{Mode: "shared"},
			},
		),
	)
})

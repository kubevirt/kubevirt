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

package storage_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/storage"
)

var _ = Describe("VirtioFS Domain Configurator", func() {
	It("Should not configure filesystems when no VirtioFS filesystems are present", func() {
		vmi := libvmi.New()
		var domain api.Domain

		Expect(storage.VirtiofsConfigurator{}.Configure(vmi, &domain)).To(Succeed())
		Expect(domain).To(Equal(api.Domain{}))
	})

	It("Should configure filesystems when VirtioFS filesystems are present", func() {
		vmi := libvmi.New(
			libvmi.WithFilesystemPVC("myfs"),
		)
		var domain api.Domain

		Expect(storage.VirtiofsConfigurator{}.Configure(vmi, &domain)).To(Succeed())

		expectedDomain := api.Domain{
			Spec: api.DomainSpec{
				Devices: api.Devices{
					Filesystems: []api.FilesystemDevice{
						{
							Type:       "mount",
							AccessMode: "passthrough",
							Driver: &api.FilesystemDriver{
								Type:  "virtiofs",
								Queue: "1024",
							},
							Source: &api.FilesystemSource{
								Socket: "/var/run/kubevirt/virtiofs-containers/myfs.sock",
							},
							Target: &api.FilesystemTarget{
								Dir: "myfs",
							},
						},
					},
				},
			},
		}
		Expect(domain).To(Equal(expectedDomain))
	})
})

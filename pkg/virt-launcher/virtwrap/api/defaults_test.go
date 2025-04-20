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

package api

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("ArchSpecificDefaults", func() {

	ginkgo.DescribeTable("should set architecture", func(arch string, targetArch string) {
		domain := &Domain{}
		NewDefaulter(arch).setDefaults_OSType(&domain.Spec.OS.Type)
		Expect(domain.Spec.OS.Type.Arch).To(Equal(targetArch))
	},
		ginkgo.Entry("to ppc64le", "ppc64le", "ppc64le"),
		ginkgo.Entry("to arm64", "arm64", "aarch64"),
		ginkgo.Entry("to x86_64", "amd64", "x86_64"),
	)

	ginkgo.DescribeTable("should set machine type and hvm domain type", func(arch string, machineType string) {
		domain := &Domain{}
		NewDefaulter(arch).setDefaults_OSType(&domain.Spec.OS.Type)
		Expect(domain.Spec.OS.Type.Machine).To(Equal(machineType))
	},
		ginkgo.Entry("to pseries", "ppc64le", "pseries"),
		ginkgo.Entry("to arm64", "arm64", "virt"),
		ginkgo.Entry("to q35", "amd64", "q35"),
	)

	ginkgo.DescribeTable("should set libvirt namespace and use QEMU as emulator", func(arch string) {
		domain := &Domain{}
		NewDefaulter(arch).setDefaults_DomainSpec(&domain.Spec)
		Expect(domain.Spec.XmlNS).To(Equal("http://libvirt.org/schemas/domain/qemu/1.0"))
		Expect(domain.Spec.Type).To(Equal("kvm"))
	},
		ginkgo.Entry("to pseries", "ppc64le"),
		ginkgo.Entry("to virt", "arm64"),
		ginkgo.Entry("to q35", "amd64"),
	)
})

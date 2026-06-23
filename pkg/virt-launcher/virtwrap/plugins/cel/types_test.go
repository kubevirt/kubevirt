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

package cel_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"libvirt.org/go/libvirtxml"

	celutil "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/plugins/cel"
)

var _ = Describe("CEL Type System", func() {
	var (
		evaluator *celutil.Evaluator
		vmi       *v1.VirtualMachineInstance
		domain    *libvirtxml.Domain
	)

	BeforeEach(func() {
		var err error
		evaluator, err = celutil.NewEvaluator()
		Expect(err).NotTo(HaveOccurred())

		vmi = &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "test-vmi"},
		}
		domain = &libvirtxml.Domain{Type: "kvm", Name: "test-vm"}
	})

	DescribeTable("type registration via NativeTypes", func(expr string, errSubstring string) {
		err := evaluator.CompileMutation(expr)
		if errSubstring == "" {
			Expect(err).NotTo(HaveOccurred())
			return
		}
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(errSubstring))
	},
		Entry("Domain type", `Domain{Title: "test"}`, ""),
		Entry("nested types", `Domain{Memory: DomainMemory{Unit: "GiB"}}`, ""),
		Entry("deeply nested types", `Domain{Devices: DomainDeviceList{Disks: [DomainDisk{Device: "disk"}]}}`, ""),
		Entry("unknown type rejected", `Bogus{Field: "test"}`, "compiling expression"),
		Entry("unknown field rejected", `Domain{Bogus: "test"}`, "compiling expression"),
	)

	Context("partial construction", func() {
		It("should only merge explicitly set fields", func() {
			result, err := evaluator.EvaluateMutation(`Domain{Title: "modified"}`, vmi, domain)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Title).To(Equal("modified"))
			Expect(result.Name).To(Equal("test-vm"))
			Expect(result.Type).To(Equal("kvm"))
		})

		It("should handle empty struct as no-op", func() {
			result, err := evaluator.EvaluateMutation(`Domain{}`, vmi, domain)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Type).To(Equal("kvm"))
			Expect(result.Name).To(Equal("test-vm"))
		})
	})

	Context("field types", func() {
		It("should handle string fields", func() {
			result, err := evaluator.EvaluateMutation(`Domain{Title: "test"}`, vmi, domain)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Title).To(Equal("test"))
		})

		It("should handle unsigned integer fields", func() {
			result, err := evaluator.EvaluateMutation(
				`Domain{Memory: DomainMemory{Value: uint(2048), Unit: "MiB"}}`, vmi, domain,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Memory.Value).To(Equal(uint(2048)))
		})

		It("should handle list fields", func() {
			result, err := evaluator.EvaluateMutation(
				`Domain{SysInfo: [DomainSysInfo{SMBIOS: DomainSysInfoSMBIOS{}}]}`, vmi, domain,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.SysInfo).To(HaveLen(1))
		})

		It("should handle nested struct fields", func() {
			result, err := evaluator.EvaluateMutation(
				`Domain{CPU: DomainCPU{Mode: "host-passthrough"}}`, vmi, domain,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.CPU).NotTo(BeNil())
			Expect(result.CPU.Mode).To(Equal("host-passthrough"))
		})
	})

	Context("xml:-tagged fields", func() {
		It("should compile expressions using xml:\"-\" fields", func() {
			Expect(evaluator.CompileMutation(
				`Domain{Devices: DomainDeviceList{Disks: [DomainDisk{Source: DomainDiskSource{File: DomainDiskSourceFile{File: "/path"}}}]}}`,
			)).To(Succeed())
		})
	})
})

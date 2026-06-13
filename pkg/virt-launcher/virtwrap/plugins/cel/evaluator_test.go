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
	"encoding/xml"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"libvirt.org/go/libvirtxml"

	celutil "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/plugins/cel"
)

var _ = Describe("CEL Evaluator", func() {
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
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-vmi",
				Namespace: "default",
				Labels: map[string]string{
					"app": "test",
				},
			},
		}

		domain = &libvirtxml.Domain{
			Type: "kvm",
			Name: "test-vm",
			Memory: &libvirtxml.DomainMemory{
				Value: 1024,
				Unit:  "MiB",
			},
			Devices: &libvirtxml.DomainDeviceList{
				Disks: []libvirtxml.DomainDisk{
					{
						Device: "disk",
						Target: &libvirtxml.DomainDiskTarget{Dev: "vda", Bus: "virtio"},
					},
				},
			},
		}
	})

	DescribeTable("condition evaluation", func(expr string, expected bool, errSubstring string) {
		result, err := evaluator.EvaluateCondition(expr, vmi, domain)
		if errSubstring != "" {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(errSubstring))
			return
		}
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(expected))
	},
		Entry("label match returns true", `vmi.Labels["app"] == "test"`, true, ""),
		Entry("label mismatch returns false", `vmi.Labels["app"] == "other"`, false, ""),
		Entry("namespace check", `vmi.Namespace == "default"`, true, ""),
		Entry("domain field check", `domainSpec.Type == "kvm"`, true, ""),
		Entry("has() on present field", `has(domainSpec.Memory)`, true, ""),
		Entry("has() on absent field", `has(domainSpec.CPU)`, false, ""),
		Entry("contains() on string field", `domainSpec.Name.contains("test")`, true, ""),
		Entry("invalid CEL syntax", `invalid @@@ syntax`, false, "compiling expression"),
		Entry("non-bool result", `vmi.Name`, false, "condition must return bool"),
	)

	Context("mutation evaluation", func() {
		It("should set a simple string field", func() {
			result, err := evaluator.EvaluateMutation(`Domain{Title: "modified"}`, vmi, domain)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Title).To(Equal("modified"))
			Expect(result.Name).To(Equal("test-vm"))
			Expect(result.Type).To(Equal("kvm"))
		})

		It("should set SysInfo with SMBIOS field", func() {
			expr := `Domain{
				SysInfo: [DomainSysInfo{
					SMBIOS: DomainSysInfoSMBIOS{}
				}]
			}`
			result, err := evaluator.EvaluateMutation(expr, vmi, domain)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.SysInfo).To(HaveLen(1))
			Expect(result.SysInfo[0].SMBIOS).NotTo(BeNil())
		})

		It("should preserve base fields not in the mutation", func() {
			result, err := evaluator.EvaluateMutation(`Domain{Title: "test"}`, vmi, domain)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Type).To(Equal("kvm"))
			Expect(result.Name).To(Equal("test-vm"))
			Expect(result.Memory).NotTo(BeNil())
			Expect(result.Memory.Unit).To(Equal("MiB"))
		})

		It("should handle empty mutation as no-op", func() {
			result, err := evaluator.EvaluateMutation(`Domain{}`, vmi, domain)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Type).To(Equal("kvm"))
			Expect(result.Name).To(Equal("test-vm"))
		})

		It("should return error for invalid expression", func() {
			_, err := evaluator.EvaluateMutation(`bad syntax !!!`, vmi, domain)
			Expect(err).To(HaveOccurred())
		})

		It("should return error when mutation returns non-Domain type", func() {
			_, err := evaluator.EvaluateMutation(`domainSpec.Name`, vmi, domain)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mutation must return a Domain object"))
		})
	})

	Context("deep merge", func() {
		It("should merge nested struct fields", func() {
			expr := `Domain{Memory: DomainMemory{Value: uint(2048), Unit: "GiB"}}`
			result, err := evaluator.EvaluateMutation(expr, vmi, domain)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Memory).NotTo(BeNil())
			Expect(result.Memory.Value).To(Equal(uint(2048)))
			Expect(result.Memory.Unit).To(Equal("GiB"))
		})

		It("should replace lists entirely", func() {
			expr := `Domain{
				SysInfo: [
					DomainSysInfo{SMBIOS: DomainSysInfoSMBIOS{}},
					DomainSysInfo{FWCfg: DomainSysInfoFWCfg{}}
				]
			}`
			result, err := evaluator.EvaluateMutation(expr, vmi, domain)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.SysInfo).To(HaveLen(2))
		})

		It("should initialize nil pointer base fields", func() {
			domain.CPU = nil
			expr := `Domain{CPU: DomainCPU{Mode: "host-passthrough"}}`
			result, err := evaluator.EvaluateMutation(expr, vmi, domain)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.CPU).NotTo(BeNil())
			Expect(result.CPU.Mode).To(Equal("host-passthrough"))
		})

		It("should produce valid XML after merge", func() {
			expr := `Domain{Title: "merged"}`
			result, err := evaluator.EvaluateMutation(expr, vmi, domain)
			Expect(err).NotTo(HaveOccurred())

			xmlBytes, err := xml.Marshal(result)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(xmlBytes)).To(ContainSubstring("merged"))
		})
	})

	DescribeTable("compile-only condition validation", func(expr string, errSubstring string) {
		err := evaluator.CompileCondition(expr)
		if errSubstring == "" {
			Expect(err).NotTo(HaveOccurred())
			return
		}
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(errSubstring))
	},
		Entry("valid condition", `vmi.Name == "test"`, ""),
		Entry("invalid CEL syntax", `@@@ invalid`, "compiling expression"),
		Entry("non-bool return type", `vmi.Name`, "condition must return bool"),
	)

	DescribeTable("compile-only mutation validation", func(expr string, errSubstring string) {
		err := evaluator.CompileMutation(expr)
		if errSubstring == "" {
			Expect(err).NotTo(HaveOccurred())
			return
		}
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(errSubstring))
	},
		Entry("valid mutation", `Domain{Title: "test"}`, ""),
		Entry("typo in type name", `Domainn{Title: "test"}`, "compiling expression"),
		Entry("unknown field", `Domain{Bogus: "test"}`, "compiling expression"),
	)
})

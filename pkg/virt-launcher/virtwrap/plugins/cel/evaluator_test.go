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

	Context("mutation evaluation", func() {
		It("should use has() to inspect optional domain fields", func() {
			expr := `Domain{
				Title: has(domainSpec.Memory) ? "memory-present" : "memory-absent",
				Description: has(domainSpec.CPU) ? "cpu-present" : "cpu-absent"
			}`
			result, err := evaluator.EvaluateMutation(expr, vmi, domain)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Title).To(Equal("memory-present"))
			Expect(result.Description).To(Equal("cpu-absent"))
		})

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

		It("should be able to use lists extension to create new domain elements", func() {
			domain.IOThreads = uint(4)

			// expression should create new DomainIOThreadIDs and populate elements for each IOThread
			expr := `Domain{IOThreadIDs: DomainIOThreadIDs{IOThreads: .lists.range(int(domainSpec.IOThreads))
				.map(i, DomainIOThread{ID: uint(i+1), PoolMin: uint(1), PoolMax: uint(1)})}}`

			result, err := evaluator.EvaluateMutation(expr, vmi, domain)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.IOThreadIDs).NotTo(BeNil())
			Expect(result.IOThreadIDs.IOThreads).NotTo(BeEmpty())
			Expect(result.IOThreadIDs.IOThreads).To(HaveLen(int(domain.IOThreads)))
			for _, t := range result.IOThreadIDs.IOThreads {
				Expect(int(*t.PoolMax)).To(Equal(1))
				Expect(int(*t.PoolMin)).To(Equal(1))
			}
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

package api

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ArchSpecificDefaults", func() {

	DescribeTable("should set architecture", func(arch string, targetArch string) {
		domain := &Domain{}
		NewDefaulter(arch).SetDefaults_OSType(&domain.Spec.OS.Type)
		Expect(domain.Spec.OS.Type.Arch).To(Equal(targetArch))
	},
		Entry("to ppc64le", "ppc64le", "ppc64le"),
		Entry("to arm64", "arm64", "aarch64"),
		Entry("to x86_64", "amd64", "x86_64"),
	)

	DescribeTable("should set machine type and hvm domain type", func(arch string, machineType string) {
		domain := &Domain{}
		NewDefaulter(arch).SetDefaults_OSType(&domain.Spec.OS.Type)
		Expect(domain.Spec.OS.Type.Machine).To(Equal(machineType))
	},
		Entry("to pseries", "ppc64le", "pseries"),
		Entry("to arm64", "arm64", "virt"),
		Entry("to q35", "amd64", "q35"),
	)

	DescribeTable("should set libvirt namespace and use QEMU as emulator", func(arch string) {
		domain := &Domain{}
		NewDefaulter(arch).SetDefaults_DomainSpec(&domain.Spec)
		Expect(domain.Spec.XmlNS).To(Equal("http://libvirt.org/schemas/domain/qemu/1.0"))
		Expect(domain.Spec.Type).To(Equal("kvm"))
	},
		Entry("to pseries", "ppc64le"),
		Entry("to virt", "arm64"),
		Entry("to q35", "amd64"),
	)
})

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
		ginkgo.Entry("to arm64", "arm64", "aarch64"),
		ginkgo.Entry("to x86_64", "amd64", "x86_64"),
	)

	ginkgo.DescribeTable("should set machine type and hvm domain type", func(arch string, machineType string) {
		domain := &Domain{}
		NewDefaulter(arch).setDefaults_OSType(&domain.Spec.OS.Type)
		Expect(domain.Spec.OS.Type.Machine).To(Equal(machineType))
	},
		ginkgo.Entry("to arm64", "arm64", "virt"),
		ginkgo.Entry("to q35", "amd64", "q35"),
	)

	ginkgo.DescribeTable("should set libvirt namespace and use QEMU as emulator", func(arch string) {
		domain := &Domain{}
		NewDefaulter(arch).setDefaults_DomainSpec(&domain.Spec)
		Expect(domain.Spec.XmlNS).To(Equal("http://libvirt.org/schemas/domain/qemu/1.0"))
		Expect(domain.Spec.Type).To(Equal("kvm"))
	},
		ginkgo.Entry("to virt", "arm64"),
		ginkgo.Entry("to q35", "amd64"),
	)
})

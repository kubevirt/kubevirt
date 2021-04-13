package api

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("ArchSpecificDefaults", func() {

	table.DescribeTable("should set architecture", func(arch string, targetArch string) {
		domain := &Domain{}
		NewDefaulter(arch).SetDefaults_OSType(&domain.Spec.OS.Type)
		Expect(domain.Spec.OS.Type.Arch).To(Equal(targetArch))
	},
		table.Entry("to ppc64le", "ppc64le", "ppc64le"),
		table.Entry("to arm64", "arm64", "aarch64"),
		table.Entry("to x86_64", "amd64", "x86_64"),
	)

	table.DescribeTable("should set machine type and hvm domain type", func(arch string, machineType string) {
		domain := &Domain{}
		NewDefaulter(arch).SetDefaults_OSType(&domain.Spec.OS.Type)
		Expect(domain.Spec.OS.Type.Machine).To(Equal(machineType))
	},
		table.Entry("to pseries", "ppc64le", "pseries"),
		table.Entry("to arm64", "arm64", "virt"),
		table.Entry("to q35", "amd64", "q35"),
	)

	table.DescribeTable("should set libvirt namespace and use QEMU as emulator", func(arch string) {
		domain := &Domain{}
		NewDefaulter(arch).SetDefaults_DomainSpec(&domain.Spec)
		Expect(domain.Spec.XmlNS).To(Equal("http://libvirt.org/schemas/domain/qemu/1.0"))
		Expect(domain.Spec.Type).To(Equal("kvm"))
	},
		table.Entry("to pseries", "ppc64le"),
		table.Entry("to virt", "arm64"),
		table.Entry("to q35", "amd64"),
	)
})

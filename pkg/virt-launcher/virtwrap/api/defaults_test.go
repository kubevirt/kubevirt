package api

import (
	"runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Defaults", func() {

	It("should set architecture", func() {
		domain := &Domain{}
		SetDefaults_OSType(&domain.Spec.OS.Type)
		if runtime.GOARCH == "ppc64le" {
			Expect(domain.Spec.OS.Type.Arch).To(Equal("ppc64le"))
		} else {
			Expect(domain.Spec.OS.Type.Arch).To(Equal("x86_64"))
		}
	})

	It("should set machine type and hvm domain type", func() {
		domain := &Domain{}
		SetDefaults_OSType(&domain.Spec.OS.Type)
		if runtime.GOARCH == "ppc64le" {
			Expect(domain.Spec.OS.Type.Machine).To(Equal("pseries"))
		} else {
			Expect(domain.Spec.OS.Type.Machine).To(Equal("q35"))
		}
		Expect(domain.Spec.OS.Type.OS).To(Equal("hvm"))
	})

	It("should set libvirt namespace and use QEMU as emulator", func() {
		domain := &Domain{}
		SetDefaults_DomainSpec(&domain.Spec)
		Expect(domain.Spec.XmlNS).To(Equal("http://libvirt.org/schemas/domain/qemu/1.0"))
		Expect(domain.Spec.Type).To(Equal("kvm"))
	})
})

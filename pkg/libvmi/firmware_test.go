package libvmi

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestWithUefiAndWithArchitectureOrderIndependence(t *testing.T) {
	RegisterTestingT(t)

	archFirst := New(WithArchitecture("arm64"), WithUefi(true))
	uefiFirst := New(WithUefi(true), WithArchitecture("arm64"))

	Expect(archFirst.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot).ToNot(BeNil())
	Expect(*archFirst.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot).To(BeTrue())
	Expect(archFirst.Spec.Domain.Features == nil || archFirst.Spec.Domain.Features.SMM == nil).To(BeTrue())

	Expect(uefiFirst.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot).ToNot(BeNil())
	Expect(*uefiFirst.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot).To(BeTrue())
	Expect(uefiFirst.Spec.Domain.Features == nil || uefiFirst.Spec.Domain.Features.SMM == nil).To(BeTrue())

	amd64First := New(WithArchitecture("amd64"), WithUefi(true))
	uefiThenAmd64 := New(WithUefi(true), WithArchitecture("amd64"))

	Expect(amd64First.Spec.Domain.Features).ToNot(BeNil())
	Expect(amd64First.Spec.Domain.Features.SMM).ToNot(BeNil())
	Expect(*amd64First.Spec.Domain.Features.SMM.Enabled).To(BeTrue())

	Expect(uefiThenAmd64.Spec.Domain.Features).ToNot(BeNil())
	Expect(uefiThenAmd64.Spec.Domain.Features.SMM).ToNot(BeNil())
	Expect(*uefiThenAmd64.Spec.Domain.Features.SMM.Enabled).To(BeTrue())
}

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

package translate

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	libvirtxml "libvirt.org/go/libvirtxml"

	"k8s.io/apimachinery/pkg/types"

	api "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Domain translation", func() {
	Context("ToLibvirtDomain", func() {
		It("should convert a minimal DomainSpec", func() {
			spec := api.NewMinimalDomainSpec("test-vm")
			spec.Type = "kvm"
			domain, err := ToLibvirtDomain(spec)
			Expect(err).ToNot(HaveOccurred())
			Expect(domain).ToNot(BeNil())
			Expect(domain.Name).To(Equal("test-vm"))
			Expect(domain.Type).To(Equal("kvm"))

			assertDomainSpecRoundTrip(spec)
		})

		It("should convert a DomainSpec with file disk", func() {
			spec := api.NewMinimalDomainSpec("test-vm")
			spec.Devices.Disks = []api.Disk{
				{
					Type:   "file",
					Device: "disk",
					Source: api.DiskSource{
						File: "/var/run/libvirt/images/disk.img",
					},
					Target: api.DiskTarget{
						Bus:    "virtio",
						Device: "vda",
					},
					Driver: &api.DiskDriver{
						Name: "qemu",
						Type: "raw",
					},
				},
			}

			domain, err := ToLibvirtDomain(spec)
			Expect(err).ToNot(HaveOccurred())
			Expect(domain.Devices.Disks).To(HaveLen(1))
			Expect(domain.Devices.Disks[0].Source.File).ToNot(BeNil())
			Expect(domain.Devices.Disks[0].Source.File.File).To(Equal("/var/run/libvirt/images/disk.img"))

			assertDomainSpecRoundTrip(spec)
		})

		It("should convert a DomainSpec with bridge interface", func() {
			spec := api.NewMinimalDomainSpec("test-vm")
			spec.Devices.Interfaces = []api.Interface{
				{
					Type: "bridge",
					Source: api.InterfaceSource{
						Bridge: "br0",
					},
					Model: &api.Model{Type: "virtio"},
					MAC:   &api.MAC{MAC: "52:54:00:00:00:01"},
				},
			}

			domain, err := ToLibvirtDomain(spec)
			Expect(err).ToNot(HaveOccurred())
			Expect(domain.Devices.Interfaces).To(HaveLen(1))
			Expect(domain.Devices.Interfaces[0].Source.Bridge).ToNot(BeNil())
			Expect(domain.Devices.Interfaces[0].Source.Bridge.Bridge).To(Equal("br0"))

			assertDomainSpecRoundTrip(spec)
		})

		It("should convert a DomainSpec with QEMU commandline", func() {
			spec := api.NewMinimalDomainSpec("test-vm")
			spec.XmlNS = "http://libvirt.org/schemas/domain/qemu/1.0"
			spec.QEMUCmd = &api.Commandline{
				QEMUArg: []api.Arg{
					{Value: "-fw_cfg"},
					{Value: "name=opt/test,string=value"},
				},
				QEMUEnv: []api.Env{
					{Name: "QEMU_TEST", Value: "1"},
				},
			}

			domain, err := ToLibvirtDomain(spec)
			Expect(err).ToNot(HaveOccurred())
			Expect(domain.QEMUCommandline).ToNot(BeNil())
			Expect(domain.QEMUCommandline.Args).To(HaveLen(2))
			Expect(domain.QEMUCommandline.Args[0].Value).To(Equal("-fw_cfg"))
			Expect(domain.QEMUCommandline.Args[1].Value).To(Equal("name=opt/test,string=value"))
			Expect(domain.QEMUCommandline.Envs).To(HaveLen(1))
			Expect(domain.QEMUCommandline.Envs[0].Name).To(Equal("QEMU_TEST"))
			Expect(domain.QEMUCommandline.Envs[0].Value).To(Equal("1"))

			assertDomainSpecRoundTrip(spec)
		})

		It("should preserve metadata through conversion", func() {
			spec := api.NewMinimalDomainSpec("test-vm")
			spec.Metadata = api.Metadata{
				KubeVirt: api.KubeVirtMetadata{
					UID: "test-uid-12345",
					GracePeriod: &api.GracePeriodMetadata{
						DeletionGracePeriodSeconds: 30,
					},
				},
			}

			domain, err := ToLibvirtDomain(spec)
			Expect(err).ToNot(HaveOccurred())
			Expect(domain.Metadata).ToNot(BeNil())
			Expect(domain.Metadata.XML).To(ContainSubstring("test-uid-12345"))
		})

		It("should handle an empty DomainSpec", func() {
			spec := &api.DomainSpec{}
			domain, err := ToLibvirtDomain(spec)
			Expect(err).ToNot(HaveOccurred())
			Expect(domain).ToNot(BeNil())
		})

		It("should handle a DomainSpec with empty devices", func() {
			spec := api.NewMinimalDomainSpec("test-vm")
			spec.Devices = api.Devices{}
			domain, err := ToLibvirtDomain(spec)
			Expect(err).ToNot(HaveOccurred())
			Expect(domain).ToNot(BeNil())
		})

		It("should return error for nil DomainSpec", func() {
			_, err := ToLibvirtDomain(nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must not be nil"))
		})
	})

	Context("FromLibvirtDomain", func() {
		It("should convert a libvirtxml Domain back to DomainSpec", func() {
			spec := api.NewMinimalDomainSpec("test-vm")
			spec.Type = "kvm"
			domain, err := ToLibvirtDomain(spec)
			Expect(err).ToNot(HaveOccurred())

			spec2, err := FromLibvirtDomain(domain)
			Expect(err).ToNot(HaveOccurred())
			Expect(spec2).ToNot(BeNil())
			Expect(spec2.Name).To(Equal("test-vm"))
			Expect(spec2.Type).To(Equal("kvm"))
		})

		It("should return error for nil libvirtxml Domain", func() {
			_, err := FromLibvirtDomain(nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must not be nil"))
		})

		It("should set XmlNS but not QEMUCmd when QEMUCommandline has no args or envs", func() {
			domain := &libvirtxml.Domain{
				Name:            "test-vm",
				QEMUCommandline: &libvirtxml.DomainQEMUCommandline{},
			}
			spec, err := FromLibvirtDomain(domain)
			Expect(err).ToNot(HaveOccurred())
			Expect(spec.XmlNS).To(Equal("http://libvirt.org/schemas/domain/qemu/1.0"))
			Expect(spec.QEMUCmd).To(BeNil())
		})

		It("should gracefully handle libvirt-only fields that have no KubeVirt equivalent", func() {
			domain := &libvirtxml.Domain{
				Name: "test-vm",
				Type: "kvm",
				QEMUCapabilities: &libvirtxml.DomainQEMUCapabilities{
					Add: []libvirtxml.DomainQEMUCapabilitiesEntry{{Name: "cap-test"}},
				},
			}
			spec, err := FromLibvirtDomain(domain)
			Expect(err).ToNot(HaveOccurred())
			Expect(spec.Name).To(Equal("test-vm"))
		})
	})

	Context("Round-trip fidelity", func() {
		It("should round-trip a DomainSpec with metadata", func() {
			spec := api.NewMinimalDomainSpec("test-vm")
			spec.Metadata = api.Metadata{
				KubeVirt: api.KubeVirtMetadata{
					UID: "test-uid-12345",
					GracePeriod: &api.GracePeriodMetadata{
						DeletionGracePeriodSeconds: 30,
					},
				},
			}

			domain, err := ToLibvirtDomain(spec)
			Expect(err).ToNot(HaveOccurred())

			roundTripped, err := FromLibvirtDomain(domain)
			Expect(err).ToNot(HaveOccurred())

			Expect(roundTripped.Metadata.KubeVirt.UID).To(Equal(types.UID("test-uid-12345")))
			Expect(roundTripped.Metadata.KubeVirt.GracePeriod).ToNot(BeNil())
			Expect(roundTripped.Metadata.KubeVirt.GracePeriod.DeletionGracePeriodSeconds).To(Equal(int64(30)))
		})

		It("should round-trip a DomainSpec with CPU topology", func() {
			spec := api.NewMinimalDomainSpec("test-vm")
			spec.CPU = api.CPU{
				Mode: "host-passthrough",
				Topology: &api.CPUTopology{
					Sockets: 1,
					Cores:   4,
					Threads: 2,
				},
			}
			spec.VCPU = &api.VCPU{
				Placement: "static",
				CPUs:      8,
			}
			assertDomainSpecRoundTrip(spec)
		})

		It("should round-trip a DomainSpec with OS and boot order", func() {
			spec := api.NewMinimalDomainSpec("test-vm")
			spec.OS = api.OS{
				Type: api.OSType{
					OS:      "hvm",
					Arch:    "x86_64",
					Machine: "q35",
				},
				BootOrder: []api.Boot{
					{Dev: "hd"},
					{Dev: "cdrom"},
				},
			}
			assertDomainSpecRoundTrip(spec)
		})

		It("should round-trip a DomainSpec with clock and timers", func() {
			spec := api.NewMinimalDomainSpec("test-vm")
			spec.Clock = &api.Clock{
				Offset: "utc",
				Timer: []api.Timer{
					{Name: "rtc", TickPolicy: "catchup", Track: "guest"},
					{Name: "pit", TickPolicy: "delay"},
					{Name: "hpet", Present: "no"},
				},
			}
			assertDomainSpecRoundTrip(spec)
		})

		It("should round-trip a DomainSpec with features", func() {
			spec := api.NewMinimalDomainSpec("test-vm")
			spec.Features = &api.Features{
				ACPI: &api.FeatureEnabled{},
				APIC: &api.FeatureEnabled{},
				SMM:  &api.FeatureEnabled{},
			}
			assertDomainSpecRoundTrip(spec)
		})

		It("should round-trip a DomainSpec with multiple devices", func() {
			spec := api.NewMinimalDomainSpec("test-vm")
			spec.Devices.Disks = []api.Disk{
				{
					Type:   "file",
					Device: "disk",
					Source: api.DiskSource{File: "/images/disk1.img"},
					Target: api.DiskTarget{Bus: "virtio", Device: "vda"},
					Driver: &api.DiskDriver{Name: "qemu", Type: "raw"},
				},
				{
					Type:     "file",
					Device:   "cdrom",
					Source:   api.DiskSource{File: "/images/cloud-init.iso"},
					Target:   api.DiskTarget{Bus: "sata", Device: "sda"},
					Driver:   &api.DiskDriver{Name: "qemu", Type: "raw"},
					ReadOnly: &api.ReadOnly{},
				},
			}
			spec.Devices.Interfaces = []api.Interface{
				{
					Type:   "bridge",
					Source: api.InterfaceSource{Bridge: "br0"},
					Model:  &api.Model{Type: "virtio"},
					MAC:    &api.MAC{MAC: "52:54:00:00:00:01"},
				},
			}
			assertDomainSpecRoundTrip(spec)
		})
	})
})

func assertDomainSpecRoundTrip(original *api.DomainSpec) {
	domain, err := ToLibvirtDomain(original)
	Expect(err).ToNot(HaveOccurred())

	roundTripped, err := FromLibvirtDomain(domain)
	Expect(err).ToNot(HaveOccurred())

	expected := original.DeepCopy()
	normalizeForComparison(expected)
	normalizeForComparison(roundTripped)

	Expect(roundTripped).To(Equal(expected))
}

func normalizeForComparison(spec *api.DomainSpec) {
	// XMLName gets set during unmarshal; normalize it
	spec.XMLName.Local = "domain"
	spec.XMLName.Space = ""

	for i := range spec.Devices.Interfaces {
		spec.Devices.Interfaces[i].XMLName.Local = ""
	}

	// KubeVirt-only fields not present in libvirt XML are lost in round-trip
	for i := range spec.Devices.Disks {
		spec.Devices.Disks[i].FilesystemOverhead = nil
		spec.Devices.Disks[i].Capacity = nil
	}
}
